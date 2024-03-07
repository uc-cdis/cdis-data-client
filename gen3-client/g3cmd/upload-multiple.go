package g3cmd

// Deprecated: Use upload instead.
import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/uc-cdis/gen3-client/gen3-client/commonUtils"
	"github.com/uc-cdis/gen3-client/gen3-client/logs"
)

func init() {
	var bucketName string
	var manifestPath string
	var uploadPath string
	var batch bool
	var numParallel int
	var forceMultipart bool
	var includeSubDirName bool

	var uploadMultipleCmd = &cobra.Command{
		Use:     "upload-multiple",
		Short:   "Upload multiple of files from a specified manifest",
		Long:    `Get presigned URLs for multiple of files specified in a manifest file and then upload all of them. Options to run multipart uploads for large files and running multiple workers to batch upload available.`,
		Example: `./gen3-client upload-multiple --profile=<profile-name> --manifest=<path-to-manifest/manifest.json> --upload-path=<path-to-file-dir/> --bucket=<bucket-name> --force-multipart=<boolean> --include-subdirname=<boolean> --batch=<boolean>`,
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("Notice: this is the upload method which requires the user to provide GUIDs. In this method files will be uploaded to specified GUIDs.\nIf your intention is to upload files without pre-existing GUIDs, consider to use \"./gen3-client upload\" instead.\n\n")

			// Instantiate interface to Gen3
			gen3Interface := NewGen3Interface()
			profileConfig = conf.ParseConfig(profile)

			host, err := gen3Interface.GetHost(&profileConfig)
			if err != nil {
				log.Fatalln("Error occurred during parsing config file for hostname: " + err.Error())
			}
			dataExplorerURL := host.Scheme + "://" + host.Host + "/explorer"

			var objects []ManifestObject

			// initialize transmission logs
			logs.InitSucceededLog(profile)
			logs.InitFailedLog(profile)
			logs.SetToBoth()
			logs.InitScoreBoard(MaxRetryCount)
			logs.InitScoreBoard(0)

			manifestFile, err := os.Open(manifestPath)
			if err != nil {
				log.Println("Failed to open manifest file")
				log.Fatalln("A valid manifest can be acquired by using the \"Download Manifest\" button on " + dataExplorerURL)
			}
			defer manifestFile.Close()

			switch {
			case strings.EqualFold(filepath.Ext(manifestPath), ".json"):
				manifestBytes, err := ioutil.ReadFile(manifestPath)
				if err != nil {
					log.Printf("Failed reading manifest %s, %v\n", manifestPath, err)
					log.Fatalln("A valid manifest can be acquired by using the \"Download Manifest\" button on " + dataExplorerURL)
				}
				err = json.Unmarshal(manifestBytes, &objects)
				if err != nil {
					log.Fatalln("Unmarshalling manifest failed with error: " + err.Error())
				}
			default:
				log.Println("Unsupported manifast format")
				log.Fatalln("A valid manifest can be acquired by using the \"Download Manifest\" button on " + dataExplorerURL)
			}

			uploadPath, err := commonUtils.GetAbsolutePath(uploadPath)
			if err != nil {
				log.Fatalf("Error when parsing file paths: " + err.Error())
			}

			filePaths := make([]string, 0)
			for _, object := range objects {
				var filePath string
				var err error

				if object.Filename != "" {
		    			// conform to fence naming convention
					filePath, err = getFullFilePath(uploadPath, object.Filename)
				} else {
					// Otherwise, here we are assuming the local filename will be the same as GUID
					filePath, err = getFullFilePath(uploadPath, object.ObjectID)
				}
				if err != nil {
					log.Println(err.Error())
					continue
				}
				filePaths = append(filePaths, filePath)
			}

			singlePartFilePaths, multipartFilePaths := separateSingleAndMultipartUploads(filePaths, forceMultipart)

			if batch {
				workers, respCh, errCh, batchFURObjects := initBatchUploadChannels(numParallel, len(singlePartFilePaths))
				for i, filePath := range singlePartFilePaths {
					fileInfo, err := ProcessFilename(uploadPath, filePath, includeSubDirName, false)
					if err != nil {
						logs.AddToFailedLog(filePath, filepath.Base(filePath), commonUtils.FileMetadata{}, "", 0, false, true)
						log.Println("Process filename error: " + err.Error())
						return
					}
					if len(batchFURObjects) < workers {
						furObject := commonUtils.FileUploadRequestObject{FilePath: fileInfo.FilePath, Filename: fileInfo.Filename, FileMetadata: fileInfo.FileMetadata, GUID: ""}
						batchFURObjects = append(batchFURObjects, furObject) //nolint:ineffassign
					} else {
						batchUpload(gen3Interface, batchFURObjects, workers, respCh, errCh, bucketName)
						batchFURObjects = make([]commonUtils.FileUploadRequestObject, 0)
						furObject := commonUtils.FileUploadRequestObject{FilePath: fileInfo.FilePath, Filename: fileInfo.Filename, FileMetadata: fileInfo.FileMetadata, GUID: ""}
						batchFURObjects = append(batchFURObjects, furObject) //nolint:ineffassign
					}
					if !forceMultipart && i == len(singlePartFilePaths)-1 { // upload remainders
						batchUpload(gen3Interface, batchFURObjects, workers, respCh, errCh, bucketName)
					}
				}
			} else {
				processSingleUploads(gen3Interface, singlePartFilePaths, bucketName, includeSubDirName, uploadPath)
			}
			if len(multipartFilePaths) > 0 {
				processMultipartUpload(gen3Interface, multipartFilePaths, bucketName, includeSubDirName, uploadPath)
			}
			if !logs.IsFailedLogMapEmpty() {
				retryUpload(logs.GetFailedLogMap())
			}
			logs.PrintScoreBoard()
			logs.CloseAll()
		},
	}

	uploadMultipleCmd.Flags().StringVar(&profile, "profile", "", "Specify profile to use")
	uploadMultipleCmd.MarkFlagRequired("profile") //nolint:errcheck
	uploadMultipleCmd.Flags().StringVar(&manifestPath, "manifest", "", "The manifest file to read from. A valid manifest can be acquired by using the \"Download Manifest\" button in Data Explorer for Common portal")
	uploadMultipleCmd.MarkFlagRequired("manifest") //nolint:errcheck
	uploadMultipleCmd.Flags().StringVar(&uploadPath, "upload-path", "", "The directory in which contains files to be uploaded")
	uploadMultipleCmd.MarkFlagRequired("upload-path") //nolint:errcheck
	uploadMultipleCmd.Flags().BoolVar(&batch, "batch", true, "Upload in parallel")
	uploadMultipleCmd.Flags().IntVar(&numParallel, "numparallel", 3, "Number of uploads to run in parallel")
	uploadMultipleCmd.Flags().StringVar(&bucketName, "bucket", "", "The bucket to which files will be uploaded. If not provided, defaults to Gen3's configured DATA_UPLOAD_BUCKET.")
	uploadMultipleCmd.Flags().BoolVar(&forceMultipart, "force-multipart", false, "Force to use multipart upload when possible (file size >= 5MB)")
	uploadMultipleCmd.Flags().BoolVar(&includeSubDirName, "include-subdirname", false, "Include subdirectory names in file name")
	RootCmd.AddCommand(uploadMultipleCmd)
}

func processSingleUploads(gen3Interface Gen3Interface, singleFilePaths []string, bucketName string, includeSubDirName bool, uploadPath string) {
	for _, filePath := range singleFilePaths {
		file, err := os.Open(filePath)
		if err != nil {
			logs.AddToFailedLog(filePath, filepath.Base(filePath), commonUtils.FileMetadata{}, "", 0, false, true)
			log.Println("File open error: " + err.Error())
			continue
		}

		startSingleFileUpload(gen3Interface, filePath, file, bucketName, includeSubDirName, uploadPath)
		file.Close()
	}
}

func startSingleFileUpload(gen3Interface Gen3Interface, filePath string, file *os.File, bucketName string, includeSubDirName bool, uploadPath string) {
	fi, err := file.Stat()
	if err != nil {
		logs.AddToFailedLog(filePath, filepath.Base(filePath), commonUtils.FileMetadata{}, "", 0, false, true)
		log.Println("File stat error for file" + fi.Name() + ", file may be missing or unreadable because of permissions.\n")
		return
	}

	fileInfo, err := ProcessFilename(uploadPath, filePath, includeSubDirName, false)
	if err != nil {
		logs.AddToFailedLog(filePath, filepath.Base(filePath), commonUtils.FileMetadata{}, "", 0, false, true)
		log.Println("Process filename error for file: " + err.Error())
		return
	}

	respURL, guid, err := GeneratePresignedURL(gen3Interface, fileInfo.Filename, fileInfo.FileMetadata, bucketName)
	if err != nil {
		logs.AddToFailedLog(fileInfo.FilePath, fileInfo.Filename, fileInfo.FileMetadata, guid, 0, false, true)
		log.Println(err.Error())
		return
	}

	logs.AddToFailedLog(fileInfo.FilePath, fileInfo.Filename, fileInfo.FileMetadata, guid, 0, false, true)

	furObject := commonUtils.FileUploadRequestObject{FilePath: fileInfo.FilePath, Filename: fileInfo.Filename, GUID: guid, PresignedURL: respURL}
	furObject, err = GenerateUploadRequest(gen3Interface, furObject, file)
	if err != nil {
		file.Close()
		log.Printf("Error occurred during request generation: %s\n", err.Error())
		return
	}

	err = uploadFile(furObject, 0)
	if err != nil {
		log.Println(err.Error())
	} else {
		logs.IncrementScore(0)
	}
	file.Close()
}

func processMultipartUpload(gen3Interface Gen3Interface, multipartFilePaths []string, bucketName string, includeSubDirName bool, uploadPath string) {
	profileConfig := conf.ParseConfig(profile)
	if profileConfig.UseShepherd == "true" ||
		profileConfig.UseShepherd == "" && commonUtils.DefaultUseShepherd == true {
		log.Fatalf("Error: Shepherd currently does not support multipart uploads. For the moment, please disable Shepherd with\n	$ gen3-client configure --profile=%v --use-shepherd=false\nand try again.\n", profile)
	}
	log.Println("Multipart uploading....")

	for _, filePath := range multipartFilePaths {
		fileInfo, err := ProcessFilename(uploadPath, filePath, includeSubDirName, false)
		if err != nil {
			logs.AddToFailedLog(filePath, filepath.Base(filePath), commonUtils.FileMetadata{}, "", 0, false, true)
			log.Println("Process filename error for file: " + err.Error())
			continue
		}
		err = multipartUpload(gen3Interface, fileInfo, 0, bucketName)
		if err != nil {
			log.Println(err.Error())
		} else {
			logs.IncrementScore(0)
		}
	}
}
