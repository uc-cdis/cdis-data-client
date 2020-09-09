package g3cmd

// Deprecated: Use upload instead.
import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/uc-cdis/gen3-client/gen3-client/commonUtils"
	"github.com/uc-cdis/gen3-client/gen3-client/jwt"
	"github.com/uc-cdis/gen3-client/gen3-client/logs"
)

func init() {
	var manifestPath string
	var uploadPath string
	var batch bool
	var numParallel int
	var workers int
	var respCh chan *http.Response
	var errCh chan error
	var batchFURObjects []commonUtils.FileUploadRequestObject

	var uploadMultipleCmd = &cobra.Command{
		Use:     "upload-multiple",
		Short:   "Upload multiple of files from a specified manifest",
		Long:    `Get presigned URLs for multiple of files specified in a manifest file and then upload all of them.`,
		Example: `./gen3-client upload-multiple --profile=<profile-name> --manifest=<path-to-manifest/manifest.json> --upload-path=<path-to-file-dir/>`,
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("Notice: this is the upload method which requires the user to provide GUIDs. In this method files will be uploaded to specified GUIDs.\nIf your intention is to upload files without pre-existing GUIDs, consider to use \"./gen3-client upload\" instead.\n\n")

			request := new(jwt.Request)
			configure := new(jwt.Configure)
			function := new(jwt.Functions)

			function.Config = configure
			function.Request = request

			host, err := function.GetHost(profile, "")
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
				json.Unmarshal(manifestBytes, &objects)
			default:
				log.Println("Unsupported manifast format")
				log.Fatalln("A valid manifest can be acquired by using the \"Download Manifest\" button on " + dataExplorerURL)
			}

			furObjects := validateObject(objects, uploadPath)

			if batch {
				workers, respCh, errCh, batchFURObjects = initBatchUploadChannels(numParallel, len(objects))
			}

			gen3Interface := NewGen3Interface()

			for i, furObject := range furObjects {
				if batch {
					if len(batchFURObjects) < workers {
						batchFURObjects = append(batchFURObjects, furObject)
					} else {
						batchUpload(gen3Interface, batchFURObjects, workers, respCh, errCh)
						batchFURObjects = make([]commonUtils.FileUploadRequestObject, 0)
						batchFURObjects = append(batchFURObjects, furObject)
					}
					if i == len(furObjects)-1 { // upload remainders
						batchUpload(gen3Interface, batchFURObjects, workers, respCh, errCh)
					}
				} else {
					file, err := os.Open(furObject.FilePath)
					if err != nil {
						log.Println("File open error: " + err.Error())
						logs.AddToFailedLog(furObject.FilePath, furObject.Filename, commonUtils.FileMetadata{}, furObject.GUID, 0, false, true)
						logs.IncrementScore(logs.ScoreBoardLen - 1)
						continue
					}
					defer file.Close()

					furObject, err := GenerateUploadRequest(furObject, file)
					if err != nil {
						file.Close()
						logs.AddToFailedLog(furObject.FilePath, furObject.Filename, commonUtils.FileMetadata{}, furObject.GUID, 0, false, true)
						logs.IncrementScore(logs.ScoreBoardLen - 1)
						log.Printf("Error occurred during request generation: %s", err.Error())
						continue
					}
					err = uploadFile(furObject, 0)
					if err != nil {
						log.Println(err.Error())
						logs.IncrementScore(logs.ScoreBoardLen - 1)
					} else {
						logs.IncrementScore(0)
					}
					file.Close()
				}
			}
			logs.PrintScoreBoard()
			logs.CloseAll()
		},
	}

	uploadMultipleCmd.Flags().StringVar(&profile, "profile", "", "Specify profile to use")
	uploadMultipleCmd.MarkFlagRequired("profile")
	uploadMultipleCmd.Flags().StringVar(&manifestPath, "manifest", "", "The manifest file to read from. A valid manifest can be acquired by using the \"Download Manifest\" button in Data Explorer for Common portal")
	uploadMultipleCmd.MarkFlagRequired("manifest")
	uploadMultipleCmd.Flags().StringVar(&uploadPath, "upload-path", "", "The directory in which contains files to be uploaded")
	uploadMultipleCmd.MarkFlagRequired("upload-path")
	uploadMultipleCmd.Flags().BoolVar(&batch, "batch", true, "Upload in parallel")
	uploadMultipleCmd.Flags().IntVar(&numParallel, "numparallel", 3, "Number of uploads to run in parallel")
	RootCmd.AddCommand(uploadMultipleCmd)
}
