package g3cmd

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/uc-cdis/gen3-client/gen3-client/logs"
	pb "gopkg.in/cheggaaa/pb.v1"
)

func retry(attempts int, filePath string, guid string, f func() error) (err error) {
	for i := 0; ; i++ {
		err = f()
		if err == nil {
			return
		}

		logs.AddToFailedLogMap(filePath, guid, "", i, false) // we don't save presigned url for multipart upload

		if i >= (attempts - 1) {
			break
		}

		time.Sleep(GetWaitTime(i))

		log.Println("retrying after error:", err)
	}
	return fmt.Errorf("after %d attempts, last error: %s", attempts, err)
}

func multipartUpload(uploadPath string, filePath string, file *os.File, includeSubDirName bool) {
	fi, err := file.Stat()
	if err != nil {
		logs.AddToFailedLogMap(filePath, "", "", 0, false)
		log.Println("File stat error for file" + fi.Name() + ", file may be missing or unreadable because of permissions.\n")
	}

	if fi.Size() > MultipartFileSizeLimit {
		logs.AddToFailedLogMap(filePath, "", "", 0, false)
		log.Println("The file size of file " + fi.Name() + " exceeds the limit allowed and cannot be uploaded. The maximum allowed file size is 5TB.\n")
	}

	uploadID, guid, filename, err := InitMultpartUpload(uploadPath, filePath, includeSubDirName)
	if err != nil {
		logs.AddToFailedLogMap(filePath, guid, "", 0, false)
		log.Println(err.Error())
		return
	}

	totalChunks := int(fi.Size() / MultipartFileChunkSize) // this casting should be safe
	if fi.Size()%MultipartFileChunkSize != 0 {
		totalChunks++
	}

	bar := pb.New64(fi.Size()).SetUnits(pb.U_BYTES).SetRefreshRate(time.Millisecond * 10).Prefix(filename + " ")
	bar.Start()
	buf := make([]byte, MultipartFileChunkSize)
	chunk := 1
	key := guid + "/" + filename
	var parts []MultipartPartObject
	for chunk <= totalChunks {
		var presignedURL string
		err = retry(MaxRetryCount, filePath, guid, func() (err error) {
			presignedURL, err = GenerateMultpartPresignedURL(key, uploadID, chunk)
			return
		})
		if err != nil {
			break
		}

		var n int
		err = retry(MaxRetryCount, filePath, guid, func() (err error) {
			n, err = file.ReadAt(buf[:cap(buf)], int64((chunk-1)*MultipartFileChunkSize))
			buf = buf[:n]
			if err == io.EOF { // finished reading
				err = nil
			}
			return
		})
		if err != nil {
			break
		}

		var etag string
		err = retry(MaxRetryCount, filePath, guid, func() (err error) {
			req, err := http.NewRequest(http.MethodPut, presignedURL, bytes.NewReader(buf))
			req.ContentLength = int64(n)
			client := &http.Client{}
			resp, err := client.Do(req)
			if err != nil {
				err = errors.New("Error occurred during upload: " + err.Error())
			}
			if resp.StatusCode != 200 {
				err = errors.New("Upload request got a non-200 response with status code " + strconv.Itoa(resp.StatusCode))
			} else if etag = resp.Header.Get("ETag"); etag == "" {
				err = errors.New("No ETag found in header")
			}
			return
		})
		if err != nil {
			break
		}

		parts = append(parts, (MultipartPartObject{PartNumber: chunk, ETag: etag}))
		bar.Add(n)
		chunk++
	}
	bar.Finish()

	if err != nil {
		logs.AddToFailedLogMap(filePath, guid, "", MaxRetryCount, false) // we don't save presigned url for multipart upload
		log.Println(err.Error())
		logs.IncrementScore(logs.ScoreBoardLen - 1)
		return
	}

	if len(parts) != totalChunks {
		logs.AddToFailedLogMap(filePath, guid, "", MaxRetryCount, false) // we don't save presigned url for multipart upload
		log.Println("Total number of received ETags doesn't match the total number of chunks")
		logs.IncrementScore(logs.ScoreBoardLen - 1)
		return
	}

	if err = CompleteMultpartUpload(key, uploadID, parts); err != nil {
		logs.AddToFailedLogMap(filePath, guid, "", MaxRetryCount, false) // we don't save presigned url for multipart upload
		log.Println(err.Error())
		logs.IncrementScore(logs.ScoreBoardLen - 1)
		return
	}

	logs.IncrementScore(0)
}
