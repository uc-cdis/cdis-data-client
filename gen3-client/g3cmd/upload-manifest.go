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

	var uploadManifestCmd = &cobra.Command{
		Use:     "upload-manifest",
		Short:   "Upload files from a specified manifest",
		Long:    `Gets a presigned URL for a file from a GUID and then uploads the specified file.`,
		Example: `./gen3-client upload-manifest --profile=<profile-name> --manifest=<path-to-manifest/manifest.json> --upload-path=<path-to-file-dir/>`,
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("Notice: this is the upload method which requires the user to provide GUIDs. In this method files will be uploaded to specified GUIDs.\nIf your intention is to upload files without pre-existing GUIDs, consider to use \"./gen3-client upload\" instead.\n\n")

			var objects []ManifestObject

			logs.InitScoreBoard(0)
			manifestFile, err := os.Open(manifestPath)
			if err != nil {
				log.Fatalf("Failed to open manifest file\n")
			}
			defer manifestFile.Close()

			switch {
			case strings.EqualFold(filepath.Ext(manifestPath), ".json"):
				manifestBytes, err := ioutil.ReadFile(manifestPath)
				if err != nil {
					log.Fatalf("Failed reading manifest %s, %v\n", manifestPath, err)
				}
				json.Unmarshal(manifestBytes, &objects)
			default:
				log.Fatalf("Unsupported manifast format")
			}

			furObjects := validateObject(objects, uploadPath)

			if batch {
				workers, respCh, errCh, batchFURObjects = initBatchUploadChannels(numParallel, len(objects))
			}

			for _, furObject := range furObjects {
				if batch {
					if len(batchFURObjects) < workers {
						batchFURObjects = append(batchFURObjects, furObject)
					} else {
						batchUpload(uploadPath, false, batchFURObjects, workers, respCh, errCh)
						batchFURObjects = make([]commonUtils.FileUploadRequestObject, 0)
						batchFURObjects = append(batchFURObjects, furObject)
					}
				} else {
					file, err := os.Open(furObject.FilePath)
					if err != nil {
						log.Println("File open error: " + err.Error())
						logs.AddToFailedLogMap(furObject.FilePath, furObject.GUID, furObject.PresignedURL, 0, false)
						logs.IncrementScore(logs.ScoreBoardLen - 1)
						continue
					}
					defer file.Close()

					furObject, err := GenerateUploadRequest(furObject, file)
					if err != nil {
						file.Close()
						logs.AddToFailedLogMap(furObject.FilePath, furObject.GUID, furObject.PresignedURL, 0, false)
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
			logs.WriteToFailedLog()
			logs.CloseAll()
			logs.PrintScoreBoard()
		},
	}

	uploadManifestCmd.Flags().StringVar(&manifestPath, "manifest", "", "The manifest file to read from")
	uploadManifestCmd.MarkFlagRequired("manifest")
	uploadManifestCmd.Flags().StringVar(&uploadPath, "upload-path", "", "The directory in which contains files to be uploaded")
	uploadManifestCmd.MarkFlagRequired("upload-path")
	uploadManifestCmd.Flags().BoolVar(&batch, "batch", true, "Upload in parallel")
	uploadManifestCmd.Flags().IntVar(&numParallel, "numparallel", 2, "Number of uploads to run in parallel")
	RootCmd.AddCommand(uploadManifestCmd)
}
