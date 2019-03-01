package g3cmd

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/spf13/cobra"
	"github.com/uc-cdis/gen3-client/gen3-client/commonUtils"
	"github.com/uc-cdis/gen3-client/gen3-client/logs"
	pb "gopkg.in/cheggaaa/pb.v1"
)

var uploadPath string
var batch bool
var numParallel int
var includeSubDirName bool

type fileInfo struct {
	filepath string
	filename string
}

func initBathUploadChannels(numParallel int, inputSliceLen int) (int, chan *http.Response, chan error, []FileUploadRequestObject) {
	workers := numParallel
	if workers < 1 || workers > inputSliceLen {
		workers = inputSliceLen
	}
	respCh := make(chan *http.Response, inputSliceLen)
	errCh := make(chan error, inputSliceLen)
	batchFURSlice := make([]FileUploadRequestObject, 0)
	return workers, respCh, errCh, batchFURSlice
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
			fmt.Println("File \"" + filePath + "\" has been found in local submission history and has be skipped for preventing duplicated submissions.")
			continue
		} else {
			logs.AddToFailedLogMap(filePath, "", true)
		}
		validatedFilePaths = append(validatedFilePaths, filePath)
		file.Close()
	}
	logs.WriteToFailedLog(false)
	return validatedFilePaths
}

func processFilename(filePath string) (fileInfo, error) {
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
			err = errors.New("Include subdirectory names will only works if the file is under at least one subdirectory.")
		}
	}
	return fileInfo{filePath, filename}, err
}

func batchUpload(furObjects []FileUploadRequestObject, workers int, respCh chan *http.Response, errCh chan error) {
	bars := make([]*pb.ProgressBar, 0)
	respURL := ""
	var err error
	var guid string

	for i := range furObjects {
		if furObjects[i].GUID == "" {
			respURL, guid, err = GeneratePresignedURL(furObjects[i].FilePath)
			if err != nil {
				errCh <- err
				logs.AddToFailedLogMap(furObjects[i].FilePath, respURL, false)
				continue
			}
			furObjects[i].PresignedURL = respURL
			furObjects[i].GUID = guid
		}
		file, err := os.Open(furObjects[i].FilePath)
		if err != nil {
			errCh <- errors.New("File open error: " + err.Error())
			logs.AddToFailedLogMap(furObjects[i].FilePath, furObjects[i].PresignedURL, false)
			continue
		}
		defer file.Close()

		furObjects[i], err = GenerateUploadRequest(furObjects[i], file)
		if err != nil {
			file.Close()
			logs.AddToFailedLogMap(furObjects[i].FilePath, furObjects[i].PresignedURL, false)
			errCh <- errors.New("Error occurred during request generation: " + err.Error())
			continue
		}
		bars = append(bars, furObjects[i].Bar)
	}

	furObjectCh := make(chan FileUploadRequestObject, len(furObjects))

	pool, err := pb.StartPool(bars...)
	if err != nil {
		errCh <- errors.New("Error occurred during starting progress bar pool: " + err.Error())
		for _, furObject := range furObjects {
			logs.AddToFailedLogMap(furObject.FilePath, furObject.PresignedURL, false)
		}
		return
	}

	client := &http.Client{}
	wg := sync.WaitGroup{}
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			for furObject := range furObjectCh {
				resp, err := client.Do(furObject.Request)
				if err != nil {
					errCh <- err
					logs.AddToFailedLogMap(furObject.FilePath, furObject.PresignedURL, true)
				} else {
					if resp.StatusCode != 200 {
						logs.AddToFailedLogMap(furObject.FilePath, furObject.PresignedURL, true)
					} else { // Succeeded
						respCh <- resp
						logs.DeleteFromFailedLogMap(furObject.FilePath, true)
						logs.WriteToSucceededLog(furObject.FilePath, furObject.GUID, true)
					}
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

func init() {
	var uploadNewCmd = &cobra.Command{
		Use:   "upload",
		Short: "upload file(s) to object storage.",
		Long:  `Gets a presigned URL for each file and then uploads the specified file(s).`,
		Example: "For uploading a single file:\n./gen3-client upload --profile=<profile-name> --upload-path=<path-to-files/data.bam>\n" +
			"For uploading all files within an folder:\n./gen3-client upload --profile=<profile-name> --upload-path=<path-to-files/folder/>\n" +
			"Can also support regex such as:\n./gen3-client upload --profile=<profile-name> --upload-path=<path-to-files/folder/*>\n" +
			"Or:\n./gen3-client upload --profile=<profile-name> --upload-path=<path-to-files/*/folder/*.bam>",
		Run: func(cmd *cobra.Command, args []string) {
			uploadPath = filepath.Clean(uploadPath)
			filePaths, err := commonUtils.ParseFilePaths(uploadPath)
			if err != nil {
				log.Fatalf("Error when parsing file paths: " + err.Error())
			}
			if len(filePaths) == 0 {
				fmt.Println("No file has been found in the provided location \"" + uploadPath + "\"")
				return
			}
			fmt.Println("\nThe following file(s) has been found in path \"" + uploadPath + "\" and will be uploaded:")
			for _, filePath := range filePaths {
				file, _ := os.Open(filePath)
				if fi, _ := file.Stat(); !fi.IsDir() {
					fmt.Println("\t" + filePath)
				}
				file.Close()
			}
			fmt.Println()

			validatedFilePaths := validateFilePath(filePaths)

			if batch {
				workers, respCh, errCh, batchFURObjects := initBathUploadChannels(numParallel, len(validatedFilePaths))
				for _, filePath := range validatedFilePaths {
					if len(batchFURObjects) < workers {
						furObject := FileUploadRequestObject{FilePath: filePath, GUID: ""}
						batchFURObjects = append(batchFURObjects, furObject)
					} else {
						batchUpload(batchFURObjects, workers, respCh, errCh)
						batchFURObjects = make([]FileUploadRequestObject, 0)
						furObject := FileUploadRequestObject{FilePath: filePath, GUID: ""}
						batchFURObjects = append(batchFURObjects, furObject)
					}
				}
				batchUpload(batchFURObjects, workers, respCh, errCh)

				if len(errCh) > 0 {
					for err := range errCh {
						if err != nil {
							fmt.Printf("Error occurred during uploading: %s\n", err.Error())
						}
					}
				}
				logs.WriteToFailedLog(false)
				fmt.Printf("%d files uploaded.\n", len(respCh))
			} else {
				for _, filePath := range validatedFilePaths {
					respURL, guid, err := GeneratePresignedURL(filePath)
					if err != nil {
						logs.AddToFailedLogMap(filePath, respURL, false)
						log.Println(err.Error())
						continue
					}
					furObject := FileUploadRequestObject{FilePath: filePath, GUID: guid, PresignedURL: respURL}
					file, err := os.Open(filePath)
					if err != nil {
						logs.AddToFailedLogMap(filePath, respURL, false)
						log.Println("File open error")
						continue
					}
					furObject, err = GenerateUploadRequest(furObject, file)
					if err != nil {
						file.Close()
						logs.AddToFailedLogMap(filePath, respURL, false)
						log.Printf("Error occurred during request generation: %s\n", err.Error())
						continue
					}
					uploadFile(furObject)
					file.Close()
				}
				logs.WriteToFailedLog(false)
			}

			if !logs.IsFailedLogMapEmpty() {
				// TODO: retransmissions
			}
		},
	}

	uploadNewCmd.Flags().StringVar(&uploadPath, "upload-path", "", "The directory or file in which contains file(s) to be uploaded")
	uploadNewCmd.MarkFlagRequired("upload-path")
	uploadNewCmd.Flags().BoolVar(&batch, "batch", false, "Upload in parallel")
	uploadNewCmd.Flags().IntVar(&numParallel, "numparallel", 3, "Number of uploads to run in parallel")
	uploadNewCmd.Flags().BoolVar(&includeSubDirName, "include-subdirname", false, "Include subdirectory names in file name")
	RootCmd.AddCommand(uploadNewCmd)
}
