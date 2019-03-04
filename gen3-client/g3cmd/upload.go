package g3cmd

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/uc-cdis/gen3-client/gen3-client/commonUtils"
	"github.com/uc-cdis/gen3-client/gen3-client/logs"
)

var uploadPath string
var batch bool
var numParallel int

type fileInfo struct {
	filepath string
	filename string
}

func init() {
	var includeSubDirName bool
	var uploadNewCmd = &cobra.Command{
		Use:   "upload",
		Short: "upload file(s) to object storage.",
		Long:  `Gets a presigned URL for each file and then uploads the specified file(s).`,
		Example: "For uploading a single file:\n./gen3-client upload --profile=<profile-name> --upload-path=<path-to-files/data.bam>\n" +
			"For uploading all files within an folder:\n./gen3-client upload --profile=<profile-name> --upload-path=<path-to-files/folder/>\n" +
			"Can also support regex such as:\n./gen3-client upload --profile=<profile-name> --upload-path=<path-to-files/folder/*>\n" +
			"Or:\n./gen3-client upload --profile=<profile-name> --upload-path=<path-to-files/*/folder/*.bam>",
		Run: func(cmd *cobra.Command, args []string) {
			logs.InitScoreBoard(MaxRetryCount)
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
			} else {
				for _, filePath := range validatedFilePaths {
					respURL, guid, filename, err := GeneratePresignedURL(filePath, includeSubDirName)
					if err != nil {
						logs.AddToFailedLogMap(filePath, respURL, false)
						logs.IncrementScore(len(logs.ScoreBoard) - 1)
						log.Println(err.Error())
						continue
					}
					furObject := FileUploadRequestObject{FilePath: filePath, FileName: filename, GUID: guid, PresignedURL: respURL}
					file, err := os.Open(filePath)
					if err != nil {
						logs.AddToFailedLogMap(filePath, respURL, false)
						logs.IncrementScore(len(logs.ScoreBoard) - 1)
						log.Println("File open error")
						continue
					}
					furObject, err = GenerateUploadRequest(furObject, file)
					if err != nil {
						file.Close()
						logs.AddToFailedLogMap(filePath, respURL, false)
						logs.IncrementScore(len(logs.ScoreBoard) - 1)
						log.Printf("Error occurred during request generation: %s\n", err.Error())
						continue
					}
					err = uploadFile(furObject)
					if err != nil {
						logs.IncrementScore(len(logs.ScoreBoard) - 1)
						log.Println(err.Error())
					} else {
						logs.IncrementScore(0)
					}
					file.Close()
				}
				logs.WriteToFailedLog(false)
			}

			if !logs.IsFailedLogMapEmpty() {
				retryUpload(logs.GetFailedLogMap(), includeSubDirName)
			}
			logs.CloseSucceededLog()
			logs.CloseFailedLog()
			logs.PrintScoreBoard()
		},
	}

	uploadNewCmd.Flags().StringVar(&uploadPath, "upload-path", "", "The directory or file in which contains file(s) to be uploaded")
	uploadNewCmd.MarkFlagRequired("upload-path")
	uploadNewCmd.Flags().BoolVar(&batch, "batch", false, "Upload in parallel")
	uploadNewCmd.Flags().IntVar(&numParallel, "numparallel", 3, "Number of uploads to run in parallel")
	uploadNewCmd.Flags().BoolVar(&includeSubDirName, "include-subdirname", false, "Include subdirectory names in file name")
	RootCmd.AddCommand(uploadNewCmd)
}
