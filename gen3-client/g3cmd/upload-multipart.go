package g3cmd

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"sort"
	"strconv"
	"sync"
	"time"

	"github.com/uc-cdis/gen3-client/gen3-client/logs"
	pb "gopkg.in/cheggaaa/pb.v1"
)

var multipartUploadLock sync.Mutex

func retry(attempts int, filePath string, guid string, f func() error) (err error) {
	for i := 0; ; i++ {
		err = f()
		if err == nil {
			return
		}

		if i >= (attempts - 1) {
			break
		}

		time.Sleep(GetWaitTime(i))

		log.Println("Retrying after error: ", err)
	}
	return fmt.Errorf("After %d attempts, last error: %s", attempts, err)
}

func multipartUpload(g3 Gen3Interface, fileInfo FileInfo, retryCount int, bucketName string) error {
	// NOTE @mpingram -- multipartUpload does not yet use the new Shepherd API
	// because Shepherd does not yet support multipart uploads.
	file, err := os.Open(fileInfo.FilePath)
	if err != nil {
		logs.AddToFailedLog(fileInfo.FilePath, fileInfo.Filename, fileInfo.FileMetadata, "", retryCount, true, true)
		err = fmt.Errorf("FAILED multipart upload for %s due to file open error: %s", fileInfo.FilePath, err.Error())
		return err
	}
	defer file.Close()

	fi, err := file.Stat()
	if err != nil {
		logs.AddToFailedLog(fileInfo.FilePath, fileInfo.Filename, fileInfo.FileMetadata, "", retryCount, true, true)
		err = fmt.Errorf("FAILED multipart upload for %s: file stat error, file may be missing or unreadable because of permissions", fileInfo.Filename)
		return err
	}

	if fi.Size() > MultipartFileSizeLimit {
		logs.AddToFailedLog(fileInfo.FilePath, fileInfo.Filename, fileInfo.FileMetadata, "", retryCount, true, true)
		err = fmt.Errorf("FAILED multipart upload for %s: the file size has exceeded the limit allowed and cannot be uploaded. The maximum allowed file size is %s", fi.Name(), FormatSize(MultipartFileSizeLimit))
		return err
	}

	uploadID, guid, err := InitMultipartUpload(g3, fileInfo.Filename, bucketName)
	if err != nil {
		logs.AddToFailedLog(fileInfo.FilePath, fileInfo.Filename, fileInfo.FileMetadata, guid, retryCount, true, true)
		err = fmt.Errorf("FAILED multipart upload for %s: %s", fileInfo.Filename, err.Error())
		return err
	}
	// update failed log with new guid
	logs.AddToFailedLog(fileInfo.FilePath, fileInfo.Filename, fileInfo.FileMetadata, guid, retryCount, true, true)

	key := guid + "/" + fileInfo.Filename
	var parts []MultipartPartObject
	numOfWorkers, numOfChunks, chunkSize := calculateChunksAndWorkers(fi.Size())
	chunkIndexCh := make(chan int, numOfChunks)
	bar := pb.New64(fi.Size()).SetUnits(pb.U_BYTES).SetRefreshRate(time.Millisecond * 10).Prefix(fileInfo.Filename + " ")
	bar.Start()

	wg := sync.WaitGroup{}
	for i := 0; i < numOfWorkers; i++ {
		wg.Add(1)
		go func() {
			buf := make([]byte, chunkSize)
			for chunkIndex := range chunkIndexCh {
				var presignedURL string
				err = retry(MaxRetryCount, fileInfo.FilePath, guid, func() (err error) {
					presignedURL, err = GenerateMultipartPresignedURL(g3, key, uploadID, chunkIndex, bucketName)
					return
				})
				if err != nil {
					logs.AddToFailedLog(fileInfo.FilePath, fileInfo.Filename, fileInfo.FileMetadata, guid, retryCount, true, true)
					log.Println(err.Error())
					continue
				}

				var n int
				err = retry(MaxRetryCount, fileInfo.FilePath, guid, func() (err error) {
					n, err = file.ReadAt(buf[:cap(buf)], int64((chunkIndex-1))*chunkSize)
					buf = buf[:n]
					if err == io.EOF { // finished reading
						err = nil
					}
					return
				})
				if err != nil {
					logs.AddToFailedLog(fileInfo.FilePath, fileInfo.Filename, fileInfo.FileMetadata, guid, retryCount, true, true)
					log.Println(err.Error())
					continue
				}

				var eTag string
				err = retry(MaxRetryCount, fileInfo.FilePath, guid, func() (err error) {
					req, err := http.NewRequest(http.MethodPut, presignedURL, bytes.NewReader(buf))
					if err != nil {
						err = errors.New("Error occurred when creating HTTP request: " + err.Error())
						return
					}
					req.ContentLength = int64(n)
					client := &http.Client{}
					resp, err := client.Do(req)
					if err != nil {
						err = errors.New("Error occurred during upload: " + err.Error())
						return
					}
					if resp.StatusCode != 200 {
						err = errors.New("Upload request got a non-200 response with status code " + strconv.Itoa(resp.StatusCode))
						return
					} else if eTag = resp.Header.Get("ETag"); eTag == "" {
						err = errors.New("No ETag found in header")
						return
					}
					return
				})
				if err != nil {
					logs.AddToFailedLog(fileInfo.FilePath, fileInfo.Filename, fileInfo.FileMetadata, guid, retryCount, true, true)
					log.Println(err.Error())
					continue
				}

				multipartUploadLock.Lock() // to avoid racing conditions
				parts = append(parts, (MultipartPartObject{PartNumber: chunkIndex, ETag: eTag}))
				bar.Add(n)
				multipartUploadLock.Unlock()
			}
			wg.Done()
		}()
	}

	for i := 1; i <= numOfChunks; i++ {
		chunkIndexCh <- i
	}
	close(chunkIndexCh)

	wg.Wait()
	bar.Finish()

	if len(parts) != numOfChunks {
		logs.AddToFailedLog(fileInfo.FilePath, fileInfo.Filename, fileInfo.FileMetadata, guid, retryCount, true, true)
		err = fmt.Errorf("FAILED multipart upload for %s: Total number of received ETags doesn't match the total number of chunks", fileInfo.Filename)
		return err
	}

	sort.Slice(parts, func(i, j int) bool {
		return parts[i].PartNumber < parts[j].PartNumber // sort parts in ascending order
	})

	if err = CompleteMultipartUpload(g3, key, uploadID, parts, bucketName); err != nil {
		logs.AddToFailedLog(fileInfo.FilePath, fileInfo.Filename, fileInfo.FileMetadata, guid, retryCount, true, true)
		err = fmt.Errorf("FAILED multipart upload for %s: %s", fileInfo.Filename, err.Error())
		return err
	}

	log.Printf("Successfully uploaded file \"%s\" to GUID %s.\n", fileInfo.FilePath, guid)
	logs.DeleteFromFailedLog(fileInfo.FilePath, true)
	logs.WriteToSucceededLog(fileInfo.FilePath, guid, true)
	return nil
}
