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
	var bucketName string
	var includeSubDirName bool
	var uploadPath string
	var batch bool
	var forceMultipart bool
	var numParallel int
	var hasMetadata bool
	var uploadCmd = &cobra.Command{
		Use:   "upload",
		Short: "Upload file(s) to object storage.",
		Long:  `Gets a presigned URL for each file and then uploads the specified file(s).`,
		Example: "For uploading a single file:\n./gen3-client upload --profile=<profile-name> --upload-path=<path-to-files/data.bam>\n" +
			"For uploading all files within an folder:\n./gen3-client upload --profile=<profile-name> --upload-path=<path-to-files/folder/>\n" +
			"Can also support regex such as:\n./gen3-client upload --profile=<profile-name> --upload-path=<path-to-files/folder/*>\n" +
			"Or:\n./gen3-client upload --profile=<profile-name> --upload-path=<path-to-files/*/folder/*.bam>\n" +
			"This command can also upload file metadata using the --metadata flag. If the --metadata flag is passed, the gen3-client will look for a file called [filename]_metadata.json in the same folder, which contains the metadata to upload.\n" +
			"For example, if uploading the file `folder/my_file.bam`, the gen3-client will look for a metadata file at `folder/my_file_metadata.json`.\n" +
			"For the format of the metadata files, see the README.",
		Run: func(cmd *cobra.Command, args []string) {
			// initialize transmission logs
			logs.InitSucceededLog(profile)
			logs.InitFailedLog(profile)
			logs.SetToBoth()
			logs.InitScoreBoard(MaxRetryCount)

			// Instantiate interface to Gen3
			gen3Interface := NewGen3Interface()
			profileConfig = conf.ParseConfig(profile)

			if hasMetadata {
				hasShepherd, err := gen3Interface.CheckForShepherdAPI(&profileConfig)
				if err != nil {
					log.Printf("WARNING: Error when checking for Shepherd API: %v", err)
				} else {
					if !hasShepherd {
						log.Fatalf("ERROR: Metadata upload (`--metadata`) is not supported in the environment you are uploading to. Double check that you are uploading to the right profile.")
					}
				}
			}

			uploadPath, _ = commonUtils.GetAbsolutePath(uploadPath)
			filePaths, err := commonUtils.ParseFilePaths(uploadPath, hasMetadata)
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

			singlepartFilePaths, multipartFilePaths := separateSingleAndMultipartUploads(filePaths, forceMultipart)

			if batch {
				workers, respCh, errCh, batchFURObjects := initBatchUploadChannels(numParallel, len(singlepartFilePaths))
				for _, filePath := range singlepartFilePaths {
					fileInfo, err := ProcessFilename(uploadPath, filePath, includeSubDirName, hasMetadata)
					if err != nil {
						logs.AddToFailedLog(filePath, filepath.Base(filePath), commonUtils.FileMetadata{}, "", 0, false, true)
						log.Println("Process filename error: " + err.Error())
						continue
					}
					if len(batchFURObjects) < workers {
						furObject := commonUtils.FileUploadRequestObject{FilePath: fileInfo.FilePath, Filename: fileInfo.Filename, FileMetadata: fileInfo.FileMetadata, GUID: ""}
						batchFURObjects = append(batchFURObjects, furObject)
					} else {
						batchUpload(gen3Interface, batchFURObjects, workers, respCh, errCh, bucketName)
						batchFURObjects = make([]commonUtils.FileUploadRequestObject, 0)
						furObject := commonUtils.FileUploadRequestObject{FilePath: fileInfo.FilePath, Filename: fileInfo.Filename, FileMetadata: fileInfo.FileMetadata, GUID: ""}
						batchFURObjects = append(batchFURObjects, furObject)
					}
				}
				batchUpload(gen3Interface, batchFURObjects, workers, respCh, errCh, bucketName)

				if len(errCh) > 0 {
					close(errCh)
					for err := range errCh {
						if err != nil {
							log.Printf("Error occurred during uploading: %s\n", err.Error())
						}
					}
				}
			} else {
				for _, filePath := range singlepartFilePaths {
					file, err := os.Open(filePath)
					if err != nil {
						logs.AddToFailedLog(filePath, filepath.Base(filePath), commonUtils.FileMetadata{}, "", 0, false, true)
						log.Println("File open error: " + err.Error())
						continue
					}

					fi, err := file.Stat()
					if err != nil {
						logs.AddToFailedLog(filePath, filepath.Base(filePath), commonUtils.FileMetadata{}, "", 0, false, true)
						log.Println("File stat error for file" + fi.Name() + ", file may be missing or unreadable because of permissions.\n")
						continue
					}
					fileInfo, err := ProcessFilename(uploadPath, filePath, includeSubDirName, hasMetadata)
					if err != nil {
						logs.AddToFailedLog(filePath, filepath.Base(filePath), commonUtils.FileMetadata{}, "", 0, false, true)
						log.Println("Process filename error for file: " + err.Error())
						continue
					}
					// The following flow is for singlepart upload flow
					respURL, guid, err := GeneratePresignedURL(gen3Interface, fileInfo.Filename, fileInfo.FileMetadata, bucketName)
					if err != nil {
						logs.AddToFailedLog(fileInfo.FilePath, fileInfo.Filename, fileInfo.FileMetadata, guid, 0, false, true)
						log.Println(err.Error())
						continue
					}
					// update failed log with new guid
					logs.AddToFailedLog(fileInfo.FilePath, fileInfo.Filename, fileInfo.FileMetadata, guid, 0, false, true)

					furObject := commonUtils.FileUploadRequestObject{FilePath: fileInfo.FilePath, Filename: fileInfo.Filename, GUID: guid, PresignedURL: respURL}
					furObject, err = GenerateUploadRequest(gen3Interface, furObject, file)
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
			}

			// multipart upload for large files here
			if len(multipartFilePaths) > 0 {
				// NOTE(@mpingram) - For the moment Shepherd doesn't support multipart uploads.
				// Throw an error if Shepherd is enabled and user attempts to multipart upload.
				processMultipartUpload(gen3Interface, multipartFilePaths, bucketName, includeSubDirName, uploadPath)
			}

			if !logs.IsFailedLogMapEmpty() {
				retryUpload(logs.GetFailedLogMap())
			}
			logs.PrintScoreBoard()
			logs.CloseAll()
		},
	}

	uploadCmd.Flags().StringVar(&profile, "profile", "", "Specify profile to use")
	uploadCmd.MarkFlagRequired("profile") //nolint:errcheck
	uploadCmd.Flags().StringVar(&uploadPath, "upload-path", "", "The directory or file in which contains file(s) to be uploaded")
	uploadCmd.MarkFlagRequired("upload-path") //nolint:errcheck
	uploadCmd.Flags().BoolVar(&batch, "batch", false, "Upload in parallel")
	uploadCmd.Flags().IntVar(&numParallel, "numparallel", 3, "Number of uploads to run in parallel")
	uploadCmd.Flags().BoolVar(&includeSubDirName, "include-subdirname", false, "Include subdirectory names in file name")
	uploadCmd.Flags().BoolVar(&forceMultipart, "force-multipart", false, "Force to use multipart upload if possible")
	uploadCmd.Flags().BoolVar(&hasMetadata, "metadata", false, "Search for and upload file metadata alongside the file")
	uploadCmd.Flags().StringVar(&bucketName, "bucket", "", "The bucket to which files will be uploaded. If not provided, defaults to Gen3's configured DATA_UPLOAD_BUCKET.")
	RootCmd.AddCommand(uploadCmd)
}
