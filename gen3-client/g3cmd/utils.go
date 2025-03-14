package g3cmd

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"math"
	"net/http"
	"net/url"
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

// go:generate mockgen -destination=./gen3-client/mocks/mock_gen3interface.go -package=mocks github.com/uc-cdis/gen3-client/gen3-client/g3cmd Gen3Interface

// ManifestObject represents an object from manifest that downloaded from windmill / data-portal
type ManifestObject struct {
	ObjectID  string `json:"object_id"`
	SubjectID string `json:"subject_id"`
	Filename  string `json:"file_name"`
	Filesize  int64  `json:"file_size"`
}

// InitRequestObject represents the payload that sends to FENCE for getting a singlepart upload presignedURL or init a multipart upload for new object file
type InitRequestObject struct {
	Filename string `json:"file_name"`
	Bucket 	 string `json:"bucket,omitempty"`
}

// ShepherdInitRequestObject represents the payload that sends to Shepherd for getting a singlepart upload presignedURL or init a multipart upload for new object file
type ShepherdInitRequestObject struct {
	Filename string `json:"file_name"`
	Authz    struct {
		Version       string   `json:"version"`
		ResourcePaths []string `json:"resource_paths"`
	} `json:"authz"`
	Aliases []string `json:"aliases"`
	// Metadata is an encoded JSON string of any arbitrary metadata the user wishes to upload.
	Metadata map[string]interface{} `json:"metadata"`
}

// MultipartUploadRequestObject represents the payload that sends to FENCE for getting a presignedURL for a part
type MultipartUploadRequestObject struct {
	Key        string `json:"key"`
	UploadID   string `json:"uploadId"`
	PartNumber int    `json:"partNumber"`
	Bucket 	   string `json:"bucket,omitempty"`
}

// MultipartCompleteRequestObject represents the payload that sends to FENCE for completeing a multipart upload
type MultipartCompleteRequestObject struct {
	Key      string                `json:"key"`
	UploadID string                `json:"uploadId"`
	Parts    []MultipartPartObject `json:"parts"`
	Bucket 	 string `json:"bucket,omitempty"`
}

// MultipartPartObject represents a part object
type MultipartPartObject struct {
	PartNumber int    `json:"PartNumber"`
	ETag       string `json:"ETag"`
}

// FileInfo is a helper struct for including subdirname as filename
type FileInfo struct {
	FilePath     string
	Filename     string
	FileMetadata commonUtils.FileMetadata
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
func InitMultipartUpload(g3 Gen3Interface, filename string, bucketName string) (string, string, error) {
	multipartInitObject := InitRequestObject{Filename: filename, Bucket: bucketName}
	objectBytes, err := json.Marshal(multipartInitObject)
	if err != nil {
		return "", "", errors.New("Error has occurred during marshalling data for multipart upload initialization, detailed error message: " + err.Error())
	}

	msg, err := g3.DoRequestWithSignedHeader(&profileConfig, commonUtils.FenceDataMultipartInitEndpoint, "application/json", objectBytes)

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
func GenerateMultipartPresignedURL(g3 Gen3Interface, key string, uploadID string, partNumber int, bucketName string) (string, error) {
	request := new(jwt.Request)
	configure := new(jwt.Configure)
	function := new(jwt.Functions)

	function.Config = configure
	function.Request = request

	multipartUploadObject := MultipartUploadRequestObject{Key: key, UploadID: uploadID, PartNumber: partNumber, Bucket: bucketName}
	objectBytes, err := json.Marshal(multipartUploadObject)
	if err != nil {
		return "", errors.New("Error has occurred during marshalling data for multipart upload presigned url generation, detailed error message: " + err.Error())
	}

	msg, err := g3.DoRequestWithSignedHeader(&profileConfig, commonUtils.FenceDataMultipartUploadEndpoint, "application/json", objectBytes)

	if err != nil {
		return "", errors.New("Error has occurred during multipart upload presigned url generation, detailed error message: " + err.Error())
	}
	if msg.PresignedURL == "" {
		return "", errors.New("Unknown error has occurred during multipart upload presigned url generation. Please check logs from Gen3 services")
	}
	return msg.PresignedURL, err
}

// CompleteMultipartUpload helps sending requests to FENCE to complete a multipart upload
func CompleteMultipartUpload(g3 Gen3Interface, key string, uploadID string, parts []MultipartPartObject, bucketName string) error {
	request := new(jwt.Request)
	configure := new(jwt.Configure)
	function := new(jwt.Functions)

	function.Config = configure
	function.Request = request

	multipartCompleteObject := MultipartCompleteRequestObject{Key: key, UploadID: uploadID, Parts: parts, Bucket: bucketName}
	objectBytes, err := json.Marshal(multipartCompleteObject)
	if err != nil {
		return errors.New("Error has occurred during marshalling data for multipart upload, detailed error message: " + err.Error())
	}

	_, err = g3.DoRequestWithSignedHeader(&profileConfig, commonUtils.FenceDataMultipartCompleteEndpoint, "application/json", objectBytes)
	if err != nil {
		return errors.New("Error has occurred during completing multipart upload, detailed error message: " + err.Error())
	}
	return nil
}

// GetDownloadResponse helps grabbing a response for downloading a file specified with GUID
func GetDownloadResponse(g3 Gen3Interface, fdrObject *commonUtils.FileDownloadResponseObject, protocolText string) error {
	// Attempt to get the file download URL from Shepherd if it's deployed in this commons,
	// otherwise fall back to Fence.
	var fileDownloadURL string
	hasShepherd, err := g3.CheckForShepherdAPI(&profileConfig)
	if err != nil {
		log.Println("Error occurred when checking for Shepherd API: " + err.Error())
		log.Println("Falling back to Indexd...")
	} else if hasShepherd {
		endPointPostfix := commonUtils.ShepherdEndpoint + "/objects/" + fdrObject.GUID + "/download"
		_, r, err := g3.GetResponse(&profileConfig, endPointPostfix, "GET", "", nil)
		if err != nil {
			return errors.New("Error occurred when getting download URL for object " + fdrObject.GUID + " from endpoint " + endPointPostfix + " . Details: " + err.Error())
		}
		defer r.Body.Close()
		if r.StatusCode != 200 {
			buf := new(bytes.Buffer)
			buf.ReadFrom(r.Body) // nolint:errcheck
			body := buf.String()
			return errors.New("Error when getting download URL at " + endPointPostfix + " for file " + fdrObject.GUID + " : Shepherd returned non-200 status code " + strconv.Itoa(r.StatusCode) + " . Request body: " + body)
		}
		// Unmarshal into json
		urlResponse := struct {
			URL string `json:"url"`
		}{}
		err = json.NewDecoder(r.Body).Decode(&urlResponse)
		if err != nil {
			return errors.New("Error occurred when getting download URL for object " + fdrObject.GUID + " from endpoint " + endPointPostfix + " . Details: " + err.Error())
		}
		fileDownloadURL = urlResponse.URL
		if fileDownloadURL == "" {
			return errors.New("Unknown error occurred when getting download URL for object " + fdrObject.GUID + " from endpoint " + endPointPostfix + " : No URL found in response body. Check the Shepherd logs")
		}
	} else {
		endPointPostfix := commonUtils.FenceDataDownloadEndpoint + "/" + fdrObject.GUID + protocolText
		msg, err := g3.DoRequestWithSignedHeader(&profileConfig, endPointPostfix, "", nil)

		if err != nil || msg.URL == "" {
			errorMsg := "Error occurred when getting download URL for object " + fdrObject.GUID
			if err != nil {
				errorMsg += "\n Details of error: " + err.Error()
			}
			return errors.New(errorMsg)
		}
		fileDownloadURL = msg.URL
	}

	// TODO: for now we don't print fdrObject.URL in error messages since it is sensitive
	// Later after we had log level we could consider for putting URL into debug logs...
	fdrObject.URL = fileDownloadURL
	if fdrObject.Range != 0 && !strings.Contains(fdrObject.URL, "X-Amz-Signature") && !strings.Contains(fdrObject.URL, "X-Goog-Signature") { // Not S3 or GS URLs and we want resume, send HEAD req first to check if server supports range
		resp, err := http.Head(fdrObject.URL)
		if err != nil {
			errorMsg := "Error occurred when sending HEAD req to URL associated with GUID " + fdrObject.GUID
			errorMsg += "\n Details of error: " + sanitizeErrorMsg(err.Error(), fdrObject.URL)
			return errors.New(errorMsg)
		}
		if resp.Header.Get("Accept-Ranges") != "bytes" { // server does not support range, download without range header
			fdrObject.Range = 0
		}
	}

	headers := map[string]string{}
	if fdrObject.Range != 0 {
		headers["Range"] = "bytes=" + strconv.FormatInt(fdrObject.Range, 10) + "-"
	}
	resp, err := g3.MakeARequest(http.MethodGet, fdrObject.URL, "", "", headers, nil, true)
	if err != nil {
		errorMsg := "Error occurred when making request to URL associated with GUID " + fdrObject.GUID
		errorMsg += "\n Details of error: " + sanitizeErrorMsg(err.Error(), fdrObject.URL)
		return errors.New(errorMsg)
	}
	if resp.StatusCode != 200 && resp.StatusCode != 206 {
		errorMsg := "Got a non-200 or non-206 response when making request to URL associated with GUID " + fdrObject.GUID
		errorMsg += "\n HTTP status code for response: " + strconv.Itoa(resp.StatusCode)
		return errors.New(errorMsg)
	}
	fdrObject.Response = resp
	return nil
}

func sanitizeErrorMsg(errorMsg string, sensitiveURL string) string {
	return strings.ReplaceAll(errorMsg, sensitiveURL, "<SENSITIVE_URL>")
}

// GeneratePresignedURL helps sending requests to Shepherd/Fence and parsing the response in order to get presigned URL for the new upload flow
func GeneratePresignedURL(g3 Gen3Interface, filename string, fileMetadata commonUtils.FileMetadata, bucketName string) (string, string, error) {
	// Attempt to get the presigned URL of this file from Shepherd if it's deployed, otherwise fall back to Fence.
	hasShepherd, err := g3.CheckForShepherdAPI(&profileConfig)
	if err != nil {
		log.Println("Error occurred when checking for Shepherd API: " + err.Error())
		log.Println("Falling back to Fence...")
	} else if hasShepherd {
		purObject := ShepherdInitRequestObject{
			Filename: filename,
			Authz: struct {
				Version       string   `json:"version"`
				ResourcePaths []string `json:"resource_paths"`
			}{
				"0",
				fileMetadata.Authz,
			},
			Aliases:  fileMetadata.Aliases,
			Metadata: fileMetadata.Metadata,
		}
		objectBytes, err := json.Marshal(purObject)
		if err != nil {
			return "", "", errors.New("Error occurred when creating upload request for file " + filename + ". Details: " + err.Error())
		}
		endPointPostfix := commonUtils.ShepherdEndpoint + "/objects"
		_, r, err := g3.GetResponse(&profileConfig, endPointPostfix, "POST", "", objectBytes)
		if err != nil {
			return "", "", errors.New("Error occurred when requesting upload URL from " + endPointPostfix + " for file " + filename + ". Details: " + err.Error())
		}
		defer r.Body.Close()
		if r.StatusCode != 201 {
			buf := new(bytes.Buffer)
			buf.ReadFrom(r.Body) // nolint:errcheck
			body := buf.String()
			return "", "", errors.New("Error when requesting upload URL at " + endPointPostfix + " for file " + filename + ": Shepherd returned non-200 status code " + strconv.Itoa(r.StatusCode) + ". Request body: " + body)
		}
		res := struct {
			GUID string `json:"guid"`
			URL  string `json:"upload_url"`
		}{}
		err = json.NewDecoder(r.Body).Decode(&res)
		if err != nil {
			return "", "", errors.New("Error occurred when creating upload URL for file " + filename + ": . Details: " + err.Error())
		}
		if res.URL == "" || res.GUID == "" {
			return "", "", errors.New("Unknown error has occurred during presigned URL or GUID generation. Please check logs from Gen3 services")
		}
		return res.URL, res.GUID, nil
	}

	// Otherwise, fall back to Fence
	purObject := InitRequestObject{Filename: filename, Bucket: bucketName}
	objectBytes, err := json.Marshal(purObject)
	if err != nil {
		return "", "", errors.New("Error occurred when marshalling object: " + err.Error())
	}
	msg, err := g3.DoRequestWithSignedHeader(&profileConfig, commonUtils.FenceDataUploadEndpoint, "application/json", objectBytes)

	if err != nil {
		return "", "", errors.New("Something went wrong. Maybe you don't have permission to upload data or Fence is misconfigured. Detailed error message: " + err.Error())
	}
	if msg.URL == "" || msg.GUID == "" {
		return "", "", errors.New("Unknown error has occurred during presigned URL or GUID generation. Please check logs from Gen3 services")
	}
	return msg.URL, msg.GUID, err
}

// GenerateUploadRequest helps preparing the HTTP request for upload and the progress bar for single part upload
func GenerateUploadRequest(g3 Gen3Interface, furObject commonUtils.FileUploadRequestObject, file *os.File) (commonUtils.FileUploadRequestObject, error) {
        if furObject.PresignedURL == "" {
               endPointPostfix := commonUtils.FenceDataUploadEndpoint + "/" + furObject.GUID + "?file_name=" + url.QueryEscape(furObject.Filename)

                // ensure bucket is set
                if furObject.Bucket != "" {
                    endPointPostfix += "&bucket=" + furObject.Bucket
                }

		msg, err := g3.DoRequestWithSignedHeader(&profileConfig, endPointPostfix, "application/json", nil)
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
			err = errors.New("io.Copy error: " + err.Error() + "\n")
		}
		if err = pw.Close(); err != nil {
			err = errors.New("Pipe writer close error: " + err.Error() + "\n")
		}
	}()
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
func DeleteRecord(g3 Gen3Interface, guid string) (string, error) {
	request := new(jwt.Request)
	configure := new(jwt.Configure)
	function := new(jwt.Functions)

	function.Config = configure
	function.Request = request

	msg, err := function.DeleteRecord(&profileConfig, guid)
	return msg, err
}

func separateSingleAndMultipartUploads(filePaths []string, forceMultipart bool) ([]string, []string) {
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
				log.Println("File \"" + filePath + "\" has been found in local submission history and has been skipped to prevent duplicated submissions.")
				return
			}
			logs.AddToFailedLog(filePath, filepath.Base(filePath), commonUtils.FileMetadata{}, "", 0, false, true)

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
func ProcessFilename(uploadPath string, filePath string, includeSubDirName bool, includeMetadata bool) (FileInfo, error) {
	var err error
	filePath, err = commonUtils.GetAbsolutePath(filePath)
	filename := filepath.Base(filePath)
	var metadata commonUtils.FileMetadata
	if includeSubDirName {
		uploadPath, err := commonUtils.GetAbsolutePath(uploadPath)
		if err == nil {
			presentPath := strings.TrimSuffix(uploadPath, commonUtils.PathSeparator+"*")
			fileInfo, err := os.Stat(presentPath)
			if err != nil {
				log.Fatal(err)
			}
			if !fileInfo.IsDir() {
				pwd, err := os.Getwd()
				if err != nil {
					log.Fatal(err)
				}
				filename = strings.TrimPrefix(presentPath, pwd)

			} else {
				filename = strings.TrimPrefix(filePath, presentPath)
			}
			filename = strings.TrimPrefix(filename, commonUtils.PathSeparator)
		}
	}
	if includeMetadata {
		// The metadata path is the file name plus '_metadata.json'
		metadataFilePath := strings.TrimSuffix(filePath, filepath.Ext(filePath)) + "_metadata.json"
		var metadataFileBytes []byte
		if _, err := os.Stat(metadataFilePath); err == nil {
			metadataFileBytes, err = ioutil.ReadFile(metadataFilePath)
			if err != nil {
				return FileInfo{}, errors.New("Error reading metadata file " + metadataFilePath + ": " + err.Error())
			}
			err := json.Unmarshal(metadataFileBytes, &metadata)
			if err != nil {
				return FileInfo{}, errors.New("Error parsing metadata file " + metadataFilePath + ": " + err.Error())
			}
		} else {
			// No metadata file was found for this file -- proceed, but warn the user.
			log.Printf("WARNING: File metadata is enabled, but could not find the metadata file %v for file %v. Execute `gen3-client upload --help` for more info on file metadata.\n", metadataFilePath, filePath)
		}
	}
	return FileInfo{filePath, filename, metadata}, err
}

func getFullFilePath(filePath string, filename string) (string, error) {
	filePath, err := commonUtils.GetAbsolutePath(filePath)
	if err != nil {
		log.Println(err)
		return "", err
	}
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

func uploadFile(furObject commonUtils.FileUploadRequestObject, retryCount int) error {
	log.Println("Uploading data ...")
	furObject.Bar.Start()

	client := &http.Client{}
	resp, err := client.Do(furObject.Request)
	if err != nil {
		logs.AddToFailedLog(furObject.FilePath, furObject.Filename, furObject.FileMetadata, furObject.GUID, retryCount, false, true)
		furObject.Bar.Finish()
		return errors.New("Error occurred during upload: " + err.Error())
	}
	if resp.StatusCode != 200 {
		logs.AddToFailedLog(furObject.FilePath, furObject.Filename, furObject.FileMetadata, furObject.GUID, retryCount, false, true)
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

func batchUpload(gen3Interface Gen3Interface, furObjects []commonUtils.FileUploadRequestObject, workers int, respCh chan *http.Response, errCh chan error, bucketName string) {
	bars := make([]*pb.ProgressBar, 0)
	respURL := ""
	var err error
	var guid string

	for i := range furObjects {
                if furObjects[i].Bucket == "" {
                    furObjects[i].Bucket = bucketName
                }
		if furObjects[i].GUID == "" {
			respURL, guid, err = GeneratePresignedURL(gen3Interface, furObjects[i].Filename, furObjects[i].FileMetadata, bucketName)
			if err != nil {
				logs.AddToFailedLog(furObjects[i].FilePath, furObjects[i].Filename, furObjects[i].FileMetadata, guid, 0, false, true)
				errCh <- err
				continue
			}
			furObjects[i].PresignedURL = respURL
			furObjects[i].GUID = guid
			// update failed log with new guid
			logs.AddToFailedLog(furObjects[i].FilePath, furObjects[i].Filename, furObjects[i].FileMetadata, guid, 0, false, true)
		}
		file, err := os.Open(furObjects[i].FilePath)
		if err != nil {
			logs.AddToFailedLog(furObjects[i].FilePath, furObjects[i].Filename, furObjects[i].FileMetadata, furObjects[i].GUID, 0, false, true)
			errCh <- errors.New("File open error: " + err.Error())
			continue
		}
		defer file.Close()

		furObjects[i], err = GenerateUploadRequest(gen3Interface, furObjects[i], file)
		if err != nil {
			file.Close()
			logs.AddToFailedLog(furObjects[i].FilePath, furObjects[i].Filename, furObjects[i].FileMetadata, furObjects[i].GUID, 0, false, true)
			errCh <- errors.New("Error occurred during request generation: " + err.Error())
			continue
		}
		bars = append(bars, furObjects[i].Bar)
	}

	furObjectCh := make(chan commonUtils.FileUploadRequestObject, len(furObjects))

	pool, err := pb.StartPool(bars...)
	if err != nil {
		for _, furObject := range furObjects {
			logs.AddToFailedLog(furObject.FilePath, furObject.Filename, furObject.FileMetadata, furObject.GUID, 0, false, true)
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
						logs.AddToFailedLog(furObject.FilePath, furObject.Filename, furObject.FileMetadata, furObject.GUID, 0, false, true)
						errCh <- err
					} else {
						if resp.StatusCode != 200 {
							logs.AddToFailedLog(furObject.FilePath, furObject.Filename, furObject.FileMetadata, furObject.GUID, 0, false, true)
						} else { // Succeeded
							respCh <- resp
							logs.DeleteFromFailedLog(furObject.FilePath, true)
							logs.WriteToSucceededLog(furObject.FilePath, furObject.GUID, true)
							logs.IncrementScore(0)
						}
					}
				} else if furObject.FilePath != "" {
					logs.AddToFailedLog(furObject.FilePath, furObject.Filename, furObject.FileMetadata, furObject.GUID, 0, false, true)
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
	err = pool.Stop()
	if err != nil {
		errCh <- errors.New("Error occurred during stopping progress bar pool: " + err.Error())
	}
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

// Gen3Interface contains methods used to make authorized http requests to Gen3 services.
type Gen3Interface interface {
	CheckPrivileges(profileConfig *jwt.Credential) (string, map[string]interface{}, error)
	CheckForShepherdAPI(profileConfig *jwt.Credential) (bool, error)
	GetResponse(profileConfig *jwt.Credential, endpointPostPrefix string, method string, contentType string, bodyBytes []byte) (string, *http.Response, error)
	DoRequestWithSignedHeader(profileConfig *jwt.Credential, endpointPostPrefix string, contentType string, bodyBytes []byte) (jwt.JsonMessage, error)
	MakeARequest(method string, apiEndpoint string, accessToken string, contentType string, headers map[string]string, body *bytes.Buffer, noTimeout bool) (*http.Response, error)
	GetHost(profileConfig *jwt.Credential) (*url.URL, error)
}

// NewGen3Interface returns a struct that contains methods used to make authorized http requests to Gen3 services.
func NewGen3Interface() Gen3Interface {
	request := new(jwt.Request)
	configure := new(jwt.Configure)
	functions := new(jwt.Functions)
	functions.Config = configure
	functions.Request = request
	gen3Interface := struct {
		*jwt.Request
		*jwt.Functions
	}{
		request,
		functions,
	}
	return gen3Interface
}
