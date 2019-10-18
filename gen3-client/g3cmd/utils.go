package g3cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/uc-cdis/gen3-client/gen3-client/commonUtils"
	"github.com/uc-cdis/gen3-client/gen3-client/logs"

	"github.com/uc-cdis/gen3-client/gen3-client/jwt"
	pb "gopkg.in/cheggaaa/pb.v1"
)

// ManifestObject represents an object from manifest that downloaded from windmill
type ManifestObject struct {
	ObjectID  string `json:"object_id"`
	SubjectID string `json:"subject_id"`
}

// InitRequestObject represents the payload that sends to FENCE for getting a singlepart upload presignedURL or init a multipart upload for new object file
type InitRequestObject struct {
	Filename string `json:"file_name"`
}

// MultipartUploadRequestObject represents the payload that sends to FENCE for getting a presignedURL for a part
type MultipartUploadRequestObject struct {
	Key        string `json:"key"`
	UploadID   string `json:"uploadId"`
	PartNumber int    `json:"partNumber"`
}

// MultipartCompleteRequestObject represents the payload that sends to FENCE for completeing a multipart upload
type MultipartCompleteRequestObject struct {
	Key      string                `json:"key"`
	UploadID string                `json:"uploadId"`
	Parts    []MultipartPartObject `json:"parts"`
}

// MultipartPartObject represents a part object
type MultipartPartObject struct {
	PartNumber int    `json:"PartNumber"`
	ETag       string `json:"ETag"`
}

// FileInfo is a helper struct for including subdirname as filename
type FileInfo struct {
	FilePath string
	Filename string
}

// RenamedOrSkippedFileInfo is a helper struct for recording renamed or skipped files
type RenamedOrSkippedFileInfo struct {
	GUID        string
	OldFilename string
	NewFilename string
}

const (
	// B is bytes
	B int64 = iota
	// KB is kilobytes
	KB int64 = 1 << (10 * iota)
	// MB is megabytes
	MB
	// GB is gigabytes
	GB
	// TB is terrabytes
	TB
)

var unitMap = map[int64]string{
	B:  "B",
	KB: "KB",
	MB: "MB",
	GB: "GB",
	TB: "TB",
}

// FileSizeLimit is the maximun single file size for non-multipart upload (5GB)
const FileSizeLimit = 5 * GB

// MultipartFileSizeLimit is the maximun single file size for multipart upload (5TB)
const MultipartFileSizeLimit = 5 * TB
const maxMultipartNumber = 10000
const minMultipartChunkSize = 5 * MB
const defaultNumOfWorkers = 10

// MaxRetryCount is the maximum retry number per record
const MaxRetryCount = 5
const maxWaitTime = 300

// InitMultipartUpload helps sending requests to FENCE to init a multipart upload
func InitMultipartUpload(filename string) (string, string, error) {
	request := new(jwt.Request)
	configure := new(jwt.Configure)
	function := new(jwt.Functions)

	function.Config = configure
	function.Request = request

	multipartInitObject := InitRequestObject{Filename: filename}
	objectBytes, err := json.Marshal(multipartInitObject)

	msg, err := function.DoRequestWithSignedHeader(profile, "", commonUtils.FenceDataMultipartInitEndpoint, "application/json", objectBytes)

	if err != nil {
		if strings.Contains(err.Error(), "404") {
			return "", "", errors.New(err.Error() + "\nPlease check to ensure FENCE version is at 2.8.0 or beyond")
		}
		return "", "", errors.New("Error has occurred during multipart upload initialization, detailed error message: " + err.Error())
	}
	if msg.UploadID == "" || msg.GUID == "" {
		return "", "", errors.New("Unknown error has occurred during multipart upload initialization. Please check logs from Gen3 services")
	}
	return msg.UploadID, msg.GUID, err
}

// GenerateMultipartPresignedURL helps sending requests to FENCE to get a presigned URL for a part during a multipart upload
func GenerateMultipartPresignedURL(key string, uploadID string, partNumber int) (string, error) {
	request := new(jwt.Request)
	configure := new(jwt.Configure)
	function := new(jwt.Functions)

	function.Config = configure
	function.Request = request

	multipartUploadObject := MultipartUploadRequestObject{Key: key, UploadID: uploadID, PartNumber: partNumber}
	objectBytes, err := json.Marshal(multipartUploadObject)

	msg, err := function.DoRequestWithSignedHeader(profile, "", commonUtils.FenceDataMultipartUploadEndpoint, "application/json", objectBytes)

	if err != nil {
		return "", errors.New("Error has occurred during multipart upload presigned url generation, detailed error message: " + err.Error())
	}
	if msg.PresignedURL == "" {
		return "", errors.New("Unknown error has occurred during multipart upload presigned url generation. Please check logs from Gen3 services")
	}
	return msg.PresignedURL, err
}

// CompleteMultipartUpload helps sending requests to FENCE to complete a multipart upload
func CompleteMultipartUpload(key string, uploadID string, parts []MultipartPartObject) error {
	request := new(jwt.Request)
	configure := new(jwt.Configure)
	function := new(jwt.Functions)

	function.Config = configure
	function.Request = request

	multipartCompleteObject := MultipartCompleteRequestObject{Key: key, UploadID: uploadID, Parts: parts}
	objectBytes, err := json.Marshal(multipartCompleteObject)

	_, err = function.DoRequestWithSignedHeader(profile, "", commonUtils.FenceDataMultipartCompleteEndpoint, "application/json", objectBytes)

	if err != nil {
		return errors.New("Error has occurred during completing multipart upload, detailed error message: " + err.Error())
	}
	return nil
}

// GetDownloadResponse helps grabbing a response for downloading a file specified with GUID
func GetDownloadResponse(fdrObject *commonUtils.FileDownloadResponseObject, protocolText string) error {
	request := new(jwt.Request)
	configure := new(jwt.Configure)
	function := new(jwt.Functions)

	function.Config = configure
	function.Request = request
	endPointPostfix := commonUtils.FenceDataDownloadEndpoint + "/" + fdrObject.GUID + protocolText
	msg, err := function.DoRequestWithSignedHeader(profile, "", endPointPostfix, "", nil)

	if err != nil || msg.URL == "" {
		errorMsg := "Error occurred when getting download URL for object " + fdrObject.GUID
		if err != nil {
			errorMsg += "\n Details of error: " + err.Error()
		}
		return errors.New(errorMsg)
	}

	fdrObject.URL = msg.URL
	if fdrObject.Range != 0 && !strings.Contains(fdrObject.URL, "X-Amz-Signature") && !strings.Contains(fdrObject.URL, "X-Goog-Signature") { // Not S3 or GS URLs and we want resume, send HEAD req first to check if server supports range
		resp, err := http.Head(fdrObject.URL)
		if err != nil {
			errorMsg := "Error occurred when sending HEAD req to URL " + fdrObject.URL
			errorMsg += "\n Details of error: " + err.Error()
			return errors.New(errorMsg)
		}
		if resp.Header.Get("Accept-Ranges") != "bytes" { // server does not support range, download without range header
			fdrObject.Range = 0
		}
	}
	req, err := http.NewRequest(http.MethodGet, fdrObject.URL, nil)
	if err != nil {
		errorMsg := "Error occurred when creating GET req for URL " + fdrObject.URL
		errorMsg += "\n Details of error: " + err.Error()
		return errors.New(errorMsg)
	}
	if fdrObject.Range != 0 {
		req.Header.Set("Range", "bytes="+strconv.FormatInt(fdrObject.Range, 10)+"-")
	}
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		errorMsg := "Error occurred when doing GET req for URL " + fdrObject.URL
		errorMsg += "\n Details of error: " + err.Error()
		return errors.New(errorMsg)
	}
	if resp.StatusCode != 200 && resp.StatusCode != 206 {
		errorMsg := "Got a non-200 or non-206 response when doing GET req for URL " + fdrObject.URL
		errorMsg += "\n HTTP status code for response: " + strconv.Itoa(resp.StatusCode)
		return errors.New(errorMsg)
	}
	fdrObject.Response = resp
	return nil
}

// GeneratePresignedURL helps sending requests to FENCE and parsing the response in order to get presigned URL for the new upload flow
func GeneratePresignedURL(filename string) (string, string, error) {
	request := new(jwt.Request)
	configure := new(jwt.Configure)
	function := new(jwt.Functions)

	function.Config = configure
	function.Request = request

	purObject := InitRequestObject{Filename: filename}
	objectBytes, err := json.Marshal(purObject)

	msg, err := function.DoRequestWithSignedHeader(profile, "", commonUtils.FenceDataUploadEndpoint, "application/json", objectBytes)

	if err != nil {
		return "", "", errors.New("You don't have permission to upload data, detailed error message: " + err.Error())
	}
	if msg.URL == "" || msg.GUID == "" {
		return "", "", errors.New("Unknown error has occurred during presigned URL or GUID generation. Please check logs from Gen3 services")
	}
	return msg.URL, msg.GUID, err
}

// GenerateUploadRequest helps preparing the HTTP request for upload and the progress bar for single part upload
func GenerateUploadRequest(furObject commonUtils.FileUploadRequestObject, file *os.File) (commonUtils.FileUploadRequestObject, error) {
	request := new(jwt.Request)
	configure := new(jwt.Configure)
	function := new(jwt.Functions)

	function.Config = configure
	function.Request = request

	if furObject.PresignedURL == "" {
		endPointPostfix := commonUtils.FenceDataUploadEndpoint + "/" + furObject.GUID
		msg, err := function.DoRequestWithSignedHeader(profile, "", endPointPostfix, "", nil)
		if err != nil && !strings.Contains(err.Error(), "No GUID found") {
			return furObject, errors.New("Upload error: " + err.Error())
		}
		if msg.URL == "" {
			return furObject, errors.New("Upload error: error in generating presigned URL for " + furObject.Filename)
		}
		furObject.PresignedURL = msg.URL
	}

	fi, err := file.Stat()
	if err != nil {
		return furObject, errors.New("File stat error for file" + furObject.Filename + ", file may be missing or unreadable because of permissions.\n")
	}

	if fi.Size() > FileSizeLimit {
		return furObject, errors.New("The file size of file " + furObject.Filename + " exceeds the limit allowed and cannot be uploaded. The maximum allowed file size is " + FormatSize(FileSizeLimit) + ".\n")
	}

	bar := pb.New64(fi.Size()).SetUnits(pb.U_BYTES).SetRefreshRate(time.Millisecond * 10).Prefix(furObject.Filename + " ")
	pr, pw := io.Pipe()

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		var writer io.Writer
		defer pw.Close()
		defer file.Close()

		writer = io.MultiWriter(pw, bar)
		if _, err = io.Copy(writer, file); err != nil {
			err = errors.New("io.Copy error: " + err.Error() + "\n")
		}
		if err = pw.Close(); err != nil {
			err = errors.New("Pipe writer close error: " + err.Error() + "\n")
		}
		wg.Done()
	}()
	wg.Wait()
	if err != nil {
		return furObject, err
	}

	req, err := http.NewRequest(http.MethodPut, furObject.PresignedURL, pr)
	req.ContentLength = fi.Size()

	furObject.Request = req
	furObject.Bar = bar

	return furObject, err
}

// DeleteRecord helps sending requests to FENCE to delete a record from INDEXD as well as its storage locations
func DeleteRecord(guid string) (string, error) {
	request := new(jwt.Request)
	configure := new(jwt.Configure)
	function := new(jwt.Functions)

	function.Config = configure
	function.Request = request

	msg, err := function.DeleteRecord(profile, "", guid)
	return msg, err
}

func validateFilePath(filePaths []string, forceMultipart bool) ([]string, []string) {
	fileSizeLimit := FileSizeLimit // 5GB
	if forceMultipart {
		fileSizeLimit = minMultipartChunkSize // 5MB
	}
	singlepartFilePaths := make([]string, 0)
	multipartFilePaths := make([]string, 0)
	for _, filePath := range filePaths {
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			log.Printf("The file you specified \"%s\" does not exist locally", filePath)
			continue
		}

		func() {
			file, err := os.Open(filePath)
			if err != nil {
				log.Println("File open error occurred when validating file path: " + err.Error())
				return
			}
			defer file.Close()

			fi, err := file.Stat()
			if err != nil {
				log.Println("File stat error occurred when validating file path: " + err.Error())
				return
			}
			if fi.IsDir() {
				return
			}

			if logs.ExistsInSucceededLog(filePath) {
				log.Println("File \"" + filePath + "\" has been found in local submission history and has been skipped for preventing duplicated submissions.")
				return
			}
			logs.AddToFailedLog(filePath, filepath.Base(filePath), "", 0, false, true)

			if fi.Size() > MultipartFileSizeLimit {
				log.Printf("The file size of %s has exceeded the limit allowed and cannot be uploaded. The maximum allowed file size is %s\n", fi.Name(), FormatSize(MultipartFileSizeLimit))
			} else if fi.Size() > int64(fileSizeLimit) {
				multipartFilePaths = append(multipartFilePaths, filePath)
			} else {
				singlepartFilePaths = append(singlepartFilePaths, filePath)
			}
		}()
	}
	return singlepartFilePaths, multipartFilePaths
}

// ProcessFilename returns an FileInfo object which has the information about the path and name to be used for upload of a file
func ProcessFilename(uploadPath string, filePath string, includeSubDirName bool) (FileInfo, error) {
	var err error
	filePath, err = commonUtils.GetAbsolutePath(filePath)
	filename := filepath.Base(filePath)
	if includeSubDirName {
		uploadPath, err = commonUtils.GetAbsolutePath(uploadPath)
		presentDirname := strings.TrimSuffix(uploadPath, commonUtils.PathSeparator+"*")
		subFilename := strings.TrimPrefix(filePath, presentDirname)
		dir, file := filepath.Split(subFilename)
		if dir != "" && dir != commonUtils.PathSeparator {
			filename = strings.TrimPrefix(subFilename, commonUtils.PathSeparator)
			filename = strings.Replace(filename, commonUtils.PathSeparator, ".", -1)
		} else {
			filename = file
		}
	}
	return FileInfo{filePath, filename}, err
}

func getFullFilePath(filePath string, filename string) (string, error) {
	filePath, err := commonUtils.GetAbsolutePath(filePath)
	fi, err := os.Stat(filePath)
	if err != nil {
		log.Println(err)
		return "", err
	}
	switch mode := fi.Mode(); {
	case mode.IsDir():
		if strings.HasSuffix(filePath, "/") {
			return filePath + filename, nil
		}
		return filePath + "/" + filename, nil
	case mode.IsRegular():
		return "", errors.New("in manifest upload mode filePath must be a dir")
	default:
		return "", errors.New("full file path creation unsuccessful")
	}
}

func validateObject(objects []ManifestObject, uploadPath string) []commonUtils.FileUploadRequestObject {
	furObjects := make([]commonUtils.FileUploadRequestObject, 0)
	for _, object := range objects {
		guid := object.ObjectID
		// Here we are assuming the local filename will be the same as GUID
		filePath, err := getFullFilePath(uploadPath, object.ObjectID)
		if err != nil {
			log.Println(err.Error())
			continue
		}

		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			log.Printf("The file you specified \"%s\" does not exist locally.\n", filePath)
			continue
		}

		furObject := commonUtils.FileUploadRequestObject{FilePath: filePath, Filename: filepath.Base(filePath), GUID: guid}
		furObjects = append(furObjects, furObject)
	}
	return furObjects
}

func uploadFile(furObject commonUtils.FileUploadRequestObject, retryCount int) error {
	log.Println("Uploading data ...")
	furObject.Bar.Start()

	client := &http.Client{}
	resp, err := client.Do(furObject.Request)
	if err != nil {
		logs.AddToFailedLog(furObject.FilePath, furObject.Filename, furObject.GUID, retryCount, false, true)
		furObject.Bar.Finish()
		return errors.New("Error occurred during upload: " + err.Error())
	}
	if resp.StatusCode != 200 {
		logs.AddToFailedLog(furObject.FilePath, furObject.Filename, furObject.GUID, retryCount, false, true)
		furObject.Bar.Finish()
		return errors.New("Upload request got a non-200 response with status code " + strconv.Itoa(resp.StatusCode))
	}
	furObject.Bar.Finish()
	log.Printf("Successfully uploaded file \"%s\" to GUID %s.\n", furObject.FilePath, furObject.GUID)
	logs.DeleteFromFailedLog(furObject.FilePath, true)
	logs.WriteToSucceededLog(furObject.FilePath, furObject.GUID, false)
	return nil
}

func getNumberOfWorkers(numParallel int, inputSliceLen int) int {
	workers := numParallel
	if workers < 1 || workers > inputSliceLen {
		workers = inputSliceLen
	}
	return workers
}
func calculateChunksAndWorkers(fileSize int64) (int, int, int64) {
	maxChunkSize := int64(math.Ceil(float64(MultipartFileSizeLimit) / float64(maxMultipartNumber)))
	var numOfChunks int
	var numOfWorkers = defaultNumOfWorkers
	var chunkSize int64
	if fileSize >= maxChunkSize {
		numOfWorkers = 1
		chunkSize = maxChunkSize
		numOfChunks = int(math.Ceil(float64(fileSize) / float64(maxChunkSize)))
	} else if fileSize > minMultipartChunkSize*defaultNumOfWorkers && fileSize < maxChunkSize {
		chunkSize = int64(math.Ceil(float64(fileSize) / float64(numOfWorkers)))
		numOfChunks = numOfWorkers
	} else {
		chunkSize = minMultipartChunkSize
		numOfWorkers = int(math.Ceil(float64(fileSize) / float64(minMultipartChunkSize)))
		numOfChunks = numOfWorkers
	}

	return numOfWorkers, numOfChunks, chunkSize
}

func initBatchUploadChannels(numParallel int, inputSliceLen int) (int, chan *http.Response, chan error, []commonUtils.FileUploadRequestObject) {
	workers := getNumberOfWorkers(numParallel, inputSliceLen)
	respCh := make(chan *http.Response, inputSliceLen)
	errCh := make(chan error, inputSliceLen)
	batchFURSlice := make([]commonUtils.FileUploadRequestObject, 0)
	return workers, respCh, errCh, batchFURSlice
}

func batchUpload(furObjects []commonUtils.FileUploadRequestObject, workers int, respCh chan *http.Response, errCh chan error) {
	bars := make([]*pb.ProgressBar, 0)
	respURL := ""
	var err error
	var guid string

	for i := range furObjects {
		if furObjects[i].GUID == "" {
			respURL, guid, err = GeneratePresignedURL(furObjects[i].Filename)
			if err != nil {
				logs.AddToFailedLog(furObjects[i].FilePath, furObjects[i].Filename, guid, 0, false, true)
				errCh <- err
				continue
			}
			furObjects[i].PresignedURL = respURL
			furObjects[i].GUID = guid
			// update failed log with new guid
			logs.AddToFailedLog(furObjects[i].FilePath, furObjects[i].Filename, guid, 0, false, true)
		}
		file, err := os.Open(furObjects[i].FilePath)
		if err != nil {
			logs.AddToFailedLog(furObjects[i].FilePath, furObjects[i].Filename, furObjects[i].GUID, 0, false, true)
			errCh <- errors.New("File open error: " + err.Error())
			continue
		}
		defer file.Close()

		furObjects[i], err = GenerateUploadRequest(furObjects[i], file)
		if err != nil {
			file.Close()
			logs.AddToFailedLog(furObjects[i].FilePath, furObjects[i].Filename, furObjects[i].GUID, 0, false, true)
			errCh <- errors.New("Error occurred during request generation: " + err.Error())
			continue
		}
		bars = append(bars, furObjects[i].Bar)
	}

	furObjectCh := make(chan commonUtils.FileUploadRequestObject, len(furObjects))

	pool, err := pb.StartPool(bars...)
	if err != nil {
		for _, furObject := range furObjects {
			logs.AddToFailedLog(furObject.FilePath, furObject.Filename, furObject.GUID, 0, false, true)
		}
		errCh <- errors.New("Error occurred during starting progress bar pool: " + err.Error())
		return
	}

	client := &http.Client{}
	wg := sync.WaitGroup{}
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			for furObject := range furObjectCh {
				if furObject.Request != nil {
					resp, err := client.Do(furObject.Request)
					if err != nil {
						logs.AddToFailedLog(furObject.FilePath, furObject.Filename, furObject.GUID, 0, false, true)
						errCh <- err
					} else {
						if resp.StatusCode != 200 {
							logs.AddToFailedLog(furObject.FilePath, furObject.Filename, furObject.GUID, 0, false, true)
						} else { // Succeeded
							respCh <- resp
							logs.DeleteFromFailedLog(furObject.FilePath, true)
							logs.WriteToSucceededLog(furObject.FilePath, furObject.GUID, true)
							logs.IncrementScore(0)
						}
					}
				} else if furObject.FilePath != "" {
					logs.AddToFailedLog(furObject.FilePath, furObject.Filename, furObject.GUID, 0, false, true)
				}
			}
			wg.Done()
		}()
	}

	for i := range furObjects {
		furObjectCh <- furObjects[i]
	}
	close(furObjectCh)

	wg.Wait()
	pool.Stop()
}

// GetWaitTime calculates the wait time for the next retry based on retry count
func GetWaitTime(retryCount int) time.Duration {
	exponentialWaitTime := math.Pow(2, float64(retryCount))
	return time.Duration(math.Min(exponentialWaitTime, float64(maxWaitTime))) * time.Second
}

// FormatSize helps to parse a int64 size into string
func FormatSize(size int64) string {
	var unitSize int64
	switch {
	case size >= TB:
		unitSize = TB
	case size >= GB:
		unitSize = GB
	case size >= MB:
		unitSize = MB
	case size >= KB:
		unitSize = KB
	default:
		unitSize = B
	}

	return fmt.Sprintf("%.1f"+unitMap[unitSize], float64(size)/float64(unitSize))
}
