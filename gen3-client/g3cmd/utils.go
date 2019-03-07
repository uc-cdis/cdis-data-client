package g3cmd

import (
	"encoding/json"
	"errors"
	"io"
	"log"
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

// PresignedURLRequestObject represents the playload that sends to fence for getting a presignedURL for new object file
type PresignedURLRequestObject struct {
	Filename string `json:"file_name"`
}

// FileInfo is a helper struct for including subdirname as filename
type FileInfo struct {
	FilePath string
	Filename string
}

// FileSizeLimit is the maximun single file size (5GB)
const FileSizeLimit = 5 * 1024 * 1024 * 1024

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
	purObject := PresignedURLRequestObject{Filename: fileinfo.Filename}
	objectBytes, err := json.Marshal(purObject)

	respURL, guid, err := function.DoRequestWithSignedHeader(profile, "", endPointPostfix, "application/json", objectBytes)

	if respURL == "" || guid == "" {
		if err != nil {
			return "", "", "", errors.New("You don't have permission to upload data, detailed error message: " + err.Error())
		}
		return "", "", "", errors.New("Unknown error has occurred during presigned URL or GUID generation. Please check logs from Gen3 services")
	}
	return respURL, guid, fileinfo.Filename, err
}

// GenerateUploadRequest helps preparing the HTTP request for upload and the progress bar
func GenerateUploadRequest(furObject commonUtils.FileUploadRequestObject, file *os.File) (commonUtils.FileUploadRequestObject, error) {
	request := new(jwt.Request)
	configure := new(jwt.Configure)
	function := new(jwt.Functions)

	function.Config = configure
	function.Request = request

	if furObject.PresignedURL == "" {
		endPointPostfix := "/user/data/upload/" + furObject.GUID
		presignedURL, _, err := function.DoRequestWithSignedHeader(profile, "", endPointPostfix, "", nil)
		if err != nil && !strings.Contains(err.Error(), "No GUID found") {
			return furObject, errors.New("Upload error: " + err.Error())
		}
		furObject.PresignedURL = presignedURL
	}

	fi, err := file.Stat()
	if err != nil {
		return furObject, errors.New("File stat error for file" + fi.Name() + ", file may be missing or unreadable because of permissions.\n")
	}

	if fi.Size() > FileSizeLimit {
		return furObject, errors.New("The file size of file " + fi.Name() + " exceeds the limit allowed and cannot be uploaded. The maximum allowed file size is 5GB.\n")
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

func validateFilePath(filePaths []string) []string {
	validatedFilePaths := make([]string, 0)
	for _, filePath := range filePaths {
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			log.Printf("The file you specified \"%s\" does not exist locally", filePath)
			continue
		}

		file, err := os.Open(filePath)
		if err != nil {
			log.Printf("File open error")
			continue
		}

		if fi, _ := file.Stat(); fi.IsDir() {
			continue
		}

		if logs.ExistsInSucceededLog(filePath) {
			log.Println("File \"" + filePath + "\" has been found in local submission history and has be skipped for preventing duplicated submissions.")
			continue
		} else {
			logs.AddToFailedLogMap(filePath, "", "", 0, true)
		}
		validatedFilePaths = append(validatedFilePaths, filePath)
		file.Close()
	}
	logs.WriteToFailedLog()
	return validatedFilePaths
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

	client := &http.Client{}
	resp, err := client.Do(furObject.Request)
	if err != nil {
		logs.AddToFailedLogMap(furObject.FilePath, furObject.GUID, furObject.PresignedURL, retryCount, false)
		logs.WriteToFailedLog()
		furObject.Bar.Finish()
		return errors.New("Error occurred during upload: " + err.Error())
	}
	if resp.StatusCode != 200 {
		logs.AddToFailedLogMap(furObject.FilePath, furObject.GUID, furObject.PresignedURL, retryCount, false)
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

func initBatchUploadChannels(numParallel int, inputSliceLen int) (int, chan *http.Response, chan error, []commonUtils.FileUploadRequestObject) {
	workers := numParallel
	if workers < 1 || workers > inputSliceLen {
		workers = inputSliceLen
	}
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
				logs.AddToFailedLogMap(furObjects[i].FilePath, guid, respURL, 0, false)
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
			logs.AddToFailedLogMap(furObjects[i].FilePath, furObjects[i].GUID, furObjects[i].PresignedURL, 0, false)
			logs.WriteToFailedLog()
			errCh <- errors.New("File open error: " + err.Error())
			continue
		}
		defer file.Close()

		furObjects[i], err = GenerateUploadRequest(furObjects[i], file)
		if err != nil {
			file.Close()
			logs.AddToFailedLogMap(furObjects[i].FilePath, furObjects[i].GUID, furObjects[i].PresignedURL, 0, false)
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
			logs.AddToFailedLogMap(furObject.FilePath, furObject.GUID, furObject.PresignedURL, 0, true)
			logs.WriteToFailedLog()
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
						logs.AddToFailedLogMap(furObject.FilePath, furObject.GUID, furObject.PresignedURL, 0, true)
						logs.WriteToFailedLog()
						errCh <- err
					} else {
						if resp.StatusCode != 200 {
							logs.AddToFailedLogMap(furObject.FilePath, furObject.GUID, furObject.PresignedURL, 0, true)
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
					logs.AddToFailedLogMap(furObject.FilePath, furObject.GUID, furObject.PresignedURL, 0, true)
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
