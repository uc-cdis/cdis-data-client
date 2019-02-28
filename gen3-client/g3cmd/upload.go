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

func initBathUploadChannels(numParallel int, inputSliceLen int) (int, chan FileUploadRequestObject, chan *http.Response, chan error, []FileUploadRequestObject) {
	workers := numParallel
	if workers < 1 || workers > inputSliceLen {
		workers = inputSliceLen
	}
	furCh := make(chan FileUploadRequestObject, workers)
	respCh := make(chan *http.Response, inputSliceLen)
	errCh := make(chan error, inputSliceLen)
	batchFURSlice := make([]FileUploadRequestObject, 0)
	return workers, furCh, respCh, errCh, batchFURSlice
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
		}
		validatedFilePaths = append(validatedFilePaths, filePath)
		file.Close()
	}
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

func batchUpload(furObjects []FileUploadRequestObject, workers int, furObjectCh chan FileUploadRequestObject, respCh chan *http.Response, errCh chan error) {
	bars := make([]*pb.ProgressBar, 0)
	respURL := ""
	var err error

	for _, furObject := range furObjects {
		if furObject.GUID == "" {
			respURL, _, err = GeneratePresignedURL(furObject.FilePath)
			if err != nil {
				errCh <- err
				continue
			}
		}
		file, err := os.Open(furObject.FilePath)
		if err != nil {
			errCh <- errors.New("File open error: " + err.Error())
			continue
		}
		defer file.Close()

		req, bar, err := GenerateUploadRequest(furObject.GUID, respURL, file)
		if err != nil {
			file.Close()
			errCh <- errors.New("Error occurred during request generation: " + err.Error())
			continue
		}
		furObject.Request = req
		bars = append(bars, bar)
	}

	pool, err := pb.StartPool(bars...)
	if err != nil {
		panic(err)
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
				} else {
					if resp.StatusCode != 200 {
						//TODO add to failed file map
					} else {
						respCh <- resp
						logs.WriteToSucceededLog(furObject.FilePath, furObject.GUID)
					}
				}
			}
			wg.Done()
		}()
	}

	for _, furObject := range furObjects {
		furObjectCh <- furObject
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
				log.Fatalf(err.Error())
			}
			if len(filePaths) == 0 {
				log.Fatalf("Error when parsing file paths, no file has been found in the provided location \"" + uploadPath + "\"")
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
				workers, furCh, respCh, errCh, batchFURObjects := initBathUploadChannels(numParallel, len(validatedFilePaths))
				for _, filePath := range validatedFilePaths {
					if len(batchFURObjects) < workers {
						furObject := FileUploadRequestObject{FilePath: filePath, GUID: ""}
						batchFURObjects = append(batchFURObjects, furObject)
					} else {
						batchUpload(batchFURObjects, workers, furCh, respCh, errCh)
						furCh = make(chan FileUploadRequestObject, workers)
						batchFURObjects = make([]FileUploadRequestObject, 0)
						furObject := FileUploadRequestObject{FilePath: filePath, GUID: ""}
						batchFURObjects = append(batchFURObjects, furObject)
					}
				}
				batchUpload(batchFURObjects, workers, furCh, respCh, errCh)

				if len(errCh) > 0 {
					for err := range errCh {
						if err != nil {
							fmt.Printf("Error: %s\n", err.Error())
						}
					}
				}
				fmt.Printf("%d files uploaded.\n", len(respCh))
			} else {
				for _, filePath := range validatedFilePaths {
					respURL, guid, err := GeneratePresignedURL(filePath)
					if err != nil {
						log.Println(err.Error())
						continue
					}
					file, err := os.Open(filePath)
					if err != nil {
						log.Println("File open error")
						continue
					}
					req, bar, err := GenerateUploadRequest("", respURL, file)
					if err != nil {
						file.Close()
						log.Printf("Error occurred during request generation: %s\n", err.Error())
						continue
					}
					uploadFile(req, bar, guid, filePath)
					file.Close()
				}
			}
			fmt.Println("Local history data updated")
		},
	}

	uploadNewCmd.Flags().StringVar(&uploadPath, "upload-path", "", "The directory or file in which contains file(s) to be uploaded")
	uploadNewCmd.MarkFlagRequired("upload-path")
	uploadNewCmd.Flags().BoolVar(&batch, "batch", false, "Upload in parallel")
	uploadNewCmd.Flags().IntVar(&numParallel, "numparallel", 3, "Number of uploads to run in parallel")
	uploadNewCmd.Flags().BoolVar(&includeSubDirName, "include-subdirname", false, "Include subdirectory names in file name")
	RootCmd.AddCommand(uploadNewCmd)
}
