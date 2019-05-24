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

	"github.com/uc-cdis/gen3-client/gen3-client/commonUtils"
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

func multipartUpload(uploadPath string, filePath string, numParallel int, includeSubDirName bool, retryCount int) error {
	file, err := os.Open(filePath)
	defer file.Close()
	if err != nil {
		logs.AddToFailedLogMap(filePath, "", retryCount, true, true)
		err = fmt.Errorf("FAILED multipart upload for %s due to file open error: %s", filePath, err.Error())
		return err
	}

	fi, err := file.Stat()
	if err != nil {
		logs.AddToFailedLogMap(filePath, "", retryCount, true, true)
		err = fmt.Errorf("FAILED multipart upload for %s: file stat error, file may be missing or unreadable because of permissions", fi.Name())
		return err
	}

	if fi.Size() > MultipartFileSizeLimit {
		logs.AddToFailedLogMap(filePath, "", retryCount, true, true)
		err = fmt.Errorf("FAILED multipart upload for %s: the file size has exceeded the limit allowed and cannot be uploaded. The maximum allowed file size is 5TB", fi.Name())
		return err
	}

	uploadID, guid, filename, err := InitMultipartUpload(uploadPath, filePath, includeSubDirName)
	if err != nil {
		logs.AddToFailedLogMap(filePath, guid, retryCount, true, true)
		err = fmt.Errorf("FAILED multipart upload for %s: %s", filename, err.Error())
		return err
	}

	key := guid + "/" + filename
	var parts []MultipartPartObject
	numOfWorkers, numOfChunks, chunkSize := calculateChunksAndWorkers(fi.Size())
	chunkIndexCh := make(chan int, numOfChunks)
	bar := pb.New64(fi.Size()).SetUnits(pb.U_BYTES).SetRefreshRate(time.Millisecond * 10).Prefix(filename + " ")
	bar.Start()

	wg := sync.WaitGroup{}
	for i := 0; i < numOfWorkers; i++ {
		wg.Add(1)
		go func() {
			buf := make([]byte, chunkSize)
			for chunkIndex := range chunkIndexCh {
				var presignedURL string
				err = retry(MaxRetryCount, filePath, guid, func() (err error) {
					presignedURL, err = GenerateMultipartPresignedURL(key, uploadID, chunkIndex)
					return
				})
				if err != nil {
					logs.AddToFailedLogMap(filePath, guid, retryCount, true, true)
					log.Println(err.Error())
					continue
				}

				var n int
				err = retry(MaxRetryCount, filePath, guid, func() (err error) {
					n, err = file.ReadAt(buf[:cap(buf)], int64((chunkIndex-1))*chunkSize)
					buf = buf[:n]
					if err == io.EOF { // finished reading
						err = nil
					}
					return
				})
				if err != nil {
					logs.AddToFailedLogMap(filePath, guid, retryCount, true, true)
					log.Println(err.Error())
					multipartUploadLock.Unlock()
					continue
				}

				var etag string
				err = retry(MaxRetryCount, filePath, guid, func() (err error) {
					req, err := http.NewRequest(http.MethodPut, presignedURL, bytes.NewReader(buf))
					req.ContentLength = int64(n)
					client := &http.Client{Timeout: commonUtils.DefaultTimeout}
					resp, err := client.Do(req)
					if err != nil {
						err = errors.New("Error occurred during upload: " + err.Error())
						return
					}
					if resp.StatusCode != 200 {
						err = errors.New("Upload request got a non-200 response with status code " + strconv.Itoa(resp.StatusCode))
						return
					} else if etag = resp.Header.Get("ETag"); etag == "" {
						err = errors.New("No ETag found in header")
						return
					}
					return
				})
				if err != nil {
					logs.AddToFailedLogMap(filePath, guid, retryCount, true, true)
					log.Println(err.Error())
					multipartUploadLock.Unlock()
					continue
				}

				multipartUploadLock.Lock() // to avoid racing conditions
				parts = append(parts, (MultipartPartObject{PartNumber: chunkIndex, ETag: etag}))
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
		logs.AddToFailedLogMap(filePath, guid, retryCount, true, true)
		err = fmt.Errorf("FAILED multipart upload for %s: Total number of received ETags doesn't match the total number of chunks", filename)
		return err
	}

	sort.Slice(parts, func(i, j int) bool {
		return parts[i].PartNumber < parts[j].PartNumber // sort parts in ascending order
	})

	if err = CompleteMultipartUpload(key, uploadID, parts); err != nil {
		logs.AddToFailedLogMap(filePath, guid, retryCount, true, true)
		err = fmt.Errorf("FAILED multipart upload for %s: %s", filename, err.Error())
		return err
	}

	log.Printf("Successfully uploaded file \"%s\" to GUID %s.\n", filePath, guid)
	logs.DeleteFromFailedLogMap(filePath, true)
	logs.WriteToSucceededLog(filePath, guid, true)
	logs.WriteToFailedLog()
	return nil
}
