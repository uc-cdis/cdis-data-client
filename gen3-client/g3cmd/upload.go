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

func init() {
	var includeSubDirName bool
	var uploadPath string
	var batch bool
	var forceMultipart bool
	var numParallel int
	var uploadCmd = &cobra.Command{
		Use:   "upload",
		Short: "Upload file(s) to object storage.",
		Long:  `Gets a presigned URL for each file and then uploads the specified file(s).`,
		Example: "For uploading a single file:\n./gen3-client upload --profile=<profile-name> --upload-path=<path-to-files/data.bam>\n" +
			"For uploading all files within an folder:\n./gen3-client upload --profile=<profile-name> --upload-path=<path-to-files/folder/>\n" +
			"Can also support regex such as:\n./gen3-client upload --profile=<profile-name> --upload-path=<path-to-files/folder/*>\n" +
			"Or:\n./gen3-client upload --profile=<profile-name> --upload-path=<path-to-files/*/folder/*.bam>",
		Run: func(cmd *cobra.Command, args []string) {
			logs.InitScoreBoard(MaxRetryCount)
			uploadPath, _ = commonUtils.GetAbsolutePath(uploadPath)
			filePaths, err := commonUtils.ParseFilePaths(uploadPath)
			if err != nil {
				log.Fatalf("Error when parsing file paths: " + err.Error())
			}
			if len(filePaths) == 0 {
				log.Println("No file has been found in the provided location \"" + uploadPath + "\"")
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

			singlepartFilePaths, multipartFilePaths := validateFilePath(filePaths, forceMultipart)

			if batch {
				workers, respCh, errCh, batchFURObjects := initBatchUploadChannels(numParallel, len(singlepartFilePaths))
				for _, filePath := range singlepartFilePaths {
					fileInfo, err := ProcessFilename(uploadPath, filePath, includeSubDirName)
					if err != nil {
						logs.AddToFailedLogMap(filePath, filepath.Base(filePath), "", 0, false, true)
						log.Println("Process filename error: " + err.Error())
						continue
					}
					if len(batchFURObjects) < workers {
						furObject := commonUtils.FileUploadRequestObject{FilePath: fileInfo.FilePath, Filename: fileInfo.Filename, GUID: ""}
						batchFURObjects = append(batchFURObjects, furObject)
					} else {
						batchUpload(batchFURObjects, workers, respCh, errCh)
						batchFURObjects = make([]commonUtils.FileUploadRequestObject, 0)
						furObject := commonUtils.FileUploadRequestObject{FilePath: fileInfo.FilePath, Filename: fileInfo.Filename, GUID: ""}
						batchFURObjects = append(batchFURObjects, furObject)
					}
				}
				batchUpload(batchFURObjects, workers, respCh, errCh)

				if len(errCh) > 0 {
					close(errCh)
					for err := range errCh {
						if err != nil {
							log.Printf("Error occurred during uploading: %s\n", err.Error())
						}
					}
				}
				logs.WriteToFailedLog()
			} else {
				for _, filePath := range singlepartFilePaths {
					file, err := os.Open(filePath)
					if err != nil {
						logs.AddToFailedLogMap(filePath, filepath.Base(filePath), "", 0, false, true)
						log.Println("File open error: " + err.Error())
						continue
					}

					fi, err := file.Stat()
					if err != nil {
						logs.AddToFailedLogMap(filePath, filepath.Base(filePath), "", 0, false, true)
						log.Println("File stat error for file" + fi.Name() + ", file may be missing or unreadable because of permissions.\n")
						continue
					}
					fileInfo, err := ProcessFilename(uploadPath, filePath, includeSubDirName)
					if err != nil {
						logs.AddToFailedLogMap(filePath, filepath.Base(filePath), "", 0, false, true)
						log.Println("Process filename error for file: " + err.Error())
						continue
					}
					// The following flow is for singlepart upload flow
					respURL, guid, err := GeneratePresignedURL(fileInfo.Filename)
					if err != nil {
						logs.AddToFailedLogMap(fileInfo.FilePath, fileInfo.Filename, guid, 0, false, true)
						log.Println(err.Error())
						continue
					}
					furObject := commonUtils.FileUploadRequestObject{FilePath: fileInfo.FilePath, Filename: fileInfo.Filename, GUID: guid, PresignedURL: respURL}
					furObject, err = GenerateUploadRequest(furObject, file)
					if err != nil {
						file.Close()
						log.Printf("Error occurred during request generation: %s\n", err.Error())
						continue
					}
					err = uploadFile(furObject, 0)
					if err != nil {
						log.Println(err.Error())
					} else {
						logs.IncrementScore(0)
					}
					file.Close()
				}
				logs.WriteToFailedLog()
			}

			// multipart upload for large files here
			if len(multipartFilePaths) > 0 {
				log.Println("Multipart uploading....")
				for _, filePath := range multipartFilePaths {
					fileInfo, err := ProcessFilename(uploadPath, filePath, includeSubDirName)
					if err != nil {
						logs.AddToFailedLogMap(filePath, filepath.Base(filePath), "", 0, false, true)
						log.Println("Process filename error for file: " + err.Error())
						continue
					}
					err = multipartUpload(fileInfo, 0)
					if err != nil {
						log.Println(err.Error())
					} else {
						logs.IncrementScore(0)
					}
				}
			}

			if !logs.IsFailedLogMapEmpty() {
				retryUpload(logs.GetFailedLogMap())
			}
			logs.CloseAll()
			logs.PrintScoreBoard()
		},
	}

	uploadCmd.Flags().StringVar(&profile, "profile", "", "Specify profile to use")
	uploadCmd.MarkFlagRequired("profile")
	uploadCmd.Flags().StringVar(&uploadPath, "upload-path", "", "The directory or file in which contains file(s) to be uploaded")
	uploadCmd.MarkFlagRequired("upload-path")
	uploadCmd.Flags().BoolVar(&batch, "batch", false, "Upload in parallel")
	uploadCmd.Flags().IntVar(&numParallel, "numparallel", 3, "Number of uploads to run in parallel")
	uploadCmd.Flags().BoolVar(&includeSubDirName, "include-subdirname", false, "Include subdirectory names in file name")
	uploadCmd.Flags().BoolVar(&forceMultipart, "force-multipart", false, "Force to use multipart upload if possible")
	RootCmd.AddCommand(uploadCmd)
}
