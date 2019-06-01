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
	"path"
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

// InitRequestObject represents the payload that sends to fence for getting a singlepart upload presignedURL or init a multipart upload for new object file
type InitRequestObject struct {
	Filename string `json:"file_name"`
}

// MultipartUploadRequestObject represents the payload that sends to fence for getting a presignedURL for a part
type MultipartUploadRequestObject struct {
	Key        string `json:"key"`
	UploadID   string `json:"uploadId"`
	PartNumber int    `json:"partNumber"`
}

// MultipartCompleteRequestObject represents the payload that sends to fence for completeing a multipart upload
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

// InitMultipartUpload helps sending requests to fence to init a multipart upload
func InitMultipartUpload(uploadPath string, filePath string, includeSubDirName bool) (string, string, string, error) {
	request := new(jwt.Request)
	configure := new(jwt.Configure)
	function := new(jwt.Functions)

	function.Config = configure
	function.Request = request

	fileinfo, err := processFilename(uploadPath, filePath, includeSubDirName)
	if err != nil {
		log.Println(err.Error())
	}
	endPointPostfix := "/user/data/multipart/init"
	multipartInitObject := InitRequestObject{Filename: fileinfo.Filename}
	objectBytes, err := json.Marshal(multipartInitObject)

	msg, err := function.DoRequestWithSignedHeader(profile, "", endPointPostfix, "application/json", objectBytes)

	if err != nil {
		return "", "", fileinfo.Filename, errors.New("You don't have permission to initialize multipart upload, detailed error message: " + err.Error())
	}
	if msg.UploadID == "" || msg.GUID == "" {
		return "", "", fileinfo.Filename, errors.New("Unknown error has occurred during multipart upload initialization. Please check logs from Gen3 services")
	}
	return msg.UploadID, msg.GUID, fileinfo.Filename, err
}

// GenerateMultipartPresignedURL helps sending requests to fence to get a presigned URL for a part during a multipart upload
func GenerateMultipartPresignedURL(key string, uploadID string, partNumber int) (string, error) {
	request := new(jwt.Request)
	configure := new(jwt.Configure)
	function := new(jwt.Functions)

	function.Config = configure
	function.Request = request

	endPointPostfix := "/user/data/multipart/upload"
	multipartUploadObject := MultipartUploadRequestObject{Key: key, UploadID: uploadID, PartNumber: partNumber}
	objectBytes, err := json.Marshal(multipartUploadObject)

	msg, err := function.DoRequestWithSignedHeader(profile, "", endPointPostfix, "application/json", objectBytes)

	if err != nil {
		return "", errors.New("You don't have permission to generate presigned url for multipart upload, detailed error message: " + err.Error())
	}
	if msg.PresignedURL == "" {
		return "", errors.New("Unknown error has occurred during multipart upload presigned url generation. Please check logs from Gen3 services")
	}
	return msg.PresignedURL, err
}

// CompleteMultipartUpload helps sending requests to fence to complete a multipart upload
func CompleteMultipartUpload(key string, uploadID string, parts []MultipartPartObject) error {
	request := new(jwt.Request)
	configure := new(jwt.Configure)
	function := new(jwt.Functions)

	function.Config = configure
	function.Request = request

	endPointPostfix := "/user/data/multipart/complete"
	multipartCompleteObject := MultipartCompleteRequestObject{Key: key, UploadID: uploadID, Parts: parts}
	objectBytes, err := json.Marshal(multipartCompleteObject)

	_, err = function.DoRequestWithSignedHeader(profile, "", endPointPostfix, "application/json", objectBytes)

	if err != nil {
		return errors.New("Error occurred during completing multipart upload, detailed error message: " + err.Error())
	}
	return nil
}

// GeneratePresignedURL helps sending requests to fence and parsing the response
func GeneratePresignedURL(uploadPath string, filePath string, includeSubDirName bool) (string, string, string, error) {
	request := new(jwt.Request)
	configure := new(jwt.Configure)
	function := new(jwt.Functions)

	function.Config = configure
	function.Request = request

	fileinfo, err := processFilename(uploadPath, filePath, includeSubDirName)
	if err != nil {
		log.Println(err.Error())
	}
	endPointPostfix := "/user/data/upload"
	purObject := InitRequestObject{Filename: fileinfo.Filename}
	objectBytes, err := json.Marshal(purObject)

	msg, err := function.DoRequestWithSignedHeader(profile, "", endPointPostfix, "application/json", objectBytes)

	if err != nil {
		return "", "", "", errors.New("You don't have permission to upload data, detailed error message: " + err.Error())
	}
	if msg.URL == "" || msg.GUID == "" {
		return "", "", "", errors.New("Unknown error has occurred during presigned URL or GUID generation. Please check logs from Gen3 services")
	}
	return msg.URL, msg.GUID, fileinfo.Filename, err
}

// GenerateUploadRequest helps preparing the HTTP request for upload and the progress bar for single part upload
func GenerateUploadRequest(furObject commonUtils.FileUploadRequestObject, file *os.File) (commonUtils.FileUploadRequestObject, error) {
	request := new(jwt.Request)
	configure := new(jwt.Configure)
	function := new(jwt.Functions)

	function.Config = configure
	function.Request = request

	if furObject.PresignedURL == "" {
		endPointPostfix := "/user/data/upload/" + furObject.GUID
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

	go func() {
		var writer io.Writer
		defer pw.Close()
		defer file.Close()

		writer = io.MultiWriter(pw, bar)
		if _, err = io.Copy(writer, file); err != nil {
			logs.WriteToFailedLog()
			log.Printf("io.Copy error: %s\n", err)
		}
		if err = pw.Close(); err != nil {
			logs.WriteToFailedLog()
			log.Printf("Pipe writer close error: %s\n", err)
		}
	}()

	req, err := http.NewRequest(http.MethodPut, furObject.PresignedURL, pr)
	req.ContentLength = fi.Size()

	furObject.Request = req
	furObject.Bar = bar

	return furObject, err
}

// DeleteRecord helps sending requests to fence to delete a record from indexd as well as its storage locations
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

		file, err := os.Open(filePath)
		defer file.Close()
		if err != nil {
			log.Printf("File open error")
			continue
		}

		fi, _ := file.Stat()

		if fi.IsDir() {
			continue
		}

		if logs.ExistsInSucceededLog(filePath) {
			log.Println("File \"" + filePath + "\" has been found in local submission history and has be skipped for preventing duplicated submissions.")
			continue
		} else {
			logs.AddToFailedLogMap(filePath, "", 0, false, true)
		}

		if fi.Size() > MultipartFileSizeLimit {
			log.Printf("The file size of %s has exceeded the limit allowed and cannot be uploaded. The maximum allowed file size is %s\n", fi.Name(), FormatSize(MultipartFileSizeLimit))
			continue
		} else if fi.Size() > int64(fileSizeLimit) {
			multipartFilePaths = append(multipartFilePaths, filePath)
		} else {
			singlepartFilePaths = append(singlepartFilePaths, filePath)
		}
	}
	logs.WriteToFailedLog()
	return singlepartFilePaths, multipartFilePaths
}

func processFilename(uploadPath string, filePath string, includeSubDirName bool) (FileInfo, error) {
	var err error
	filename := filepath.Base(filePath)
	if includeSubDirName {
		presentDirname := strings.TrimSuffix(commonUtils.ParseRootPath(uploadPath), commonUtils.PathSeparator+"*")
		subFilename := strings.TrimPrefix(filePath, presentDirname)
		dir, _ := filepath.Split(subFilename)
		if dir != "" && dir != commonUtils.PathSeparator {
			filename = strings.TrimPrefix(subFilename, commonUtils.PathSeparator)
			filename = strings.Replace(filename, commonUtils.PathSeparator, ".", -1)
		} else {
			err = errors.New("Include subdirectory names will only works if the file is under at least one subdirectory")
		}
	}
	return FileInfo{filePath, filename}, err
}

func getFullFilePath(filePath string, filename string) (string, error) {
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

		furObject := commonUtils.FileUploadRequestObject{FilePath: filePath, Filename: path.Base(filePath), GUID: guid}
		furObjects = append(furObjects, furObject)
	}
	return furObjects
}

func uploadFile(furObject commonUtils.FileUploadRequestObject, retryCount int) error {
	log.Println("Uploading data ...")
	furObject.Bar.Start()

	client := &http.Client{Timeout: commonUtils.UploadTimeout}
	resp, err := client.Do(furObject.Request)
	if err != nil {
		logs.AddToFailedLogMap(furObject.FilePath, furObject.GUID, retryCount, false, true)
		logs.WriteToFailedLog()
		furObject.Bar.Finish()
		return errors.New("Error occurred during upload: " + err.Error())
	}
	if resp.StatusCode != 200 {
		logs.AddToFailedLogMap(furObject.FilePath, furObject.GUID, retryCount, false, true)
		logs.WriteToFailedLog()
		furObject.Bar.Finish()
		return errors.New("Upload request got a non-200 response with status code " + strconv.Itoa(resp.StatusCode))
	}
	furObject.Bar.Finish()
	log.Printf("Successfully uploaded file \"%s\" to GUID %s.\n", furObject.FilePath, furObject.GUID)
	logs.DeleteFromFailedLogMap(furObject.FilePath, true)
	logs.WriteToSucceededLog(furObject.FilePath, furObject.GUID, false)
	logs.WriteToFailedLog()
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

func batchUpload(uploadPath string, includeSubDirName bool, furObjects []commonUtils.FileUploadRequestObject, workers int, respCh chan *http.Response, errCh chan error) {
	bars := make([]*pb.ProgressBar, 0)
	respURL := ""
	var err error
	var guid string
	var filename string

	for i := range furObjects {
		if furObjects[i].GUID == "" {
			respURL, guid, filename, err = GeneratePresignedURL(uploadPath, furObjects[i].FilePath, includeSubDirName)
			if err != nil {
				logs.AddToFailedLogMap(furObjects[i].FilePath, guid, 0, false, true)
				logs.WriteToFailedLog()
				errCh <- err
				continue
			}
			furObjects[i].PresignedURL = respURL
			furObjects[i].GUID = guid
			furObjects[i].Filename = filename
		}
		file, err := os.Open(furObjects[i].FilePath)
		if err != nil {
			logs.AddToFailedLogMap(furObjects[i].FilePath, furObjects[i].GUID, 0, false, true)
			logs.WriteToFailedLog()
			errCh <- errors.New("File open error: " + err.Error())
			continue
		}
		defer file.Close()

		furObjects[i], err = GenerateUploadRequest(furObjects[i], file)
		if err != nil {
			file.Close()
			logs.AddToFailedLogMap(furObjects[i].FilePath, furObjects[i].GUID, 0, false, true)
			logs.WriteToFailedLog()
			errCh <- errors.New("Error occurred during request generation: " + err.Error())
			continue
		}
		bars = append(bars, furObjects[i].Bar)
	}

	furObjectCh := make(chan commonUtils.FileUploadRequestObject, len(furObjects))

	pool, err := pb.StartPool(bars...)
	if err != nil {
		for _, furObject := range furObjects {
			logs.AddToFailedLogMap(furObject.FilePath, furObject.GUID, 0, false, true)
			logs.WriteToFailedLog()
		}
		errCh <- errors.New("Error occurred during starting progress bar pool: " + err.Error())
		return
	}

	client := &http.Client{Timeout: commonUtils.UploadTimeout}
	wg := sync.WaitGroup{}
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			for furObject := range furObjectCh {
				if furObject.Request != nil {
					resp, err := client.Do(furObject.Request)
					if err != nil {
						logs.AddToFailedLogMap(furObject.FilePath, furObject.GUID, 0, false, true)
						logs.WriteToFailedLog()
						errCh <- err
					} else {
						if resp.StatusCode != 200 {
							logs.AddToFailedLogMap(furObject.FilePath, furObject.GUID, 0, false, true)
							logs.WriteToFailedLog()
						} else { // Succeeded
							respCh <- resp
							logs.DeleteFromFailedLogMap(furObject.FilePath, true)
							logs.WriteToSucceededLog(furObject.FilePath, furObject.GUID, true)
							logs.WriteToFailedLog()
							logs.IncrementScore(0)
						}
					}
				} else if furObject.FilePath != "" {
					logs.AddToFailedLogMap(furObject.FilePath, furObject.GUID, 0, false, true)
					logs.WriteToFailedLog()
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
