package g3cmd

// Deprecated: Use upload instead.
import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"path"
	"strconv"

	"github.com/uc-cdis/gen3-client/gen3-client/logs"

	"github.com/spf13/cobra"
	"github.com/uc-cdis/gen3-client/gen3-client/commonUtils"
)

func uploadFile(furObject FileUploadRequestObject) error {
	fmt.Println("Uploading data ...")
	furObject.Bar.Start()

	client := &http.Client{}
	resp, err := client.Do(furObject.Request)
	if err != nil {
		logs.AddToFailedLogMap(furObject.FilePath, furObject.PresignedURL, false)
		furObject.Bar.Finish()
		return errors.New("Error occurred during upload: " + err.Error())
	}
	if resp.StatusCode != 200 {
		logs.AddToFailedLogMap(furObject.FilePath, furObject.PresignedURL, false)
		furObject.Bar.Finish()
		return errors.New("Upload request got a non-200 response with status code " + strconv.Itoa(resp.StatusCode))
	}
	furObject.Bar.Finish()
	fmt.Printf("Successfully uploaded file \"%s\" to GUID %s.\n", furObject.FilePath, furObject.GUID)
	logs.DeleteFromFailedLogMap(furObject.FilePath, true)
	logs.WriteToSucceededLog(furObject.FilePath, furObject.GUID, false)
	return nil
}

func init() {
	var guid string
	var filePath string

	var uploadCmd = &cobra.Command{
		Use:     "upload-single",
		Short:   "Upload a single file to a GUID",
		Long:    `Gets a presigned URL for which to upload a file associated with a GUID and then uploads the specified file.`,
		Example: `./gen3-client upload-single --profile=<profile-name> --guid=f6923cf3-xxxx-xxxx-xxxx-14ab3f84f9d6 --file=<path-to-file>`,
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("Notice: this is the upload method which requires the user to provide a GUID. In this method file will be uploaded to a specified GUID.\nIf your intention is to upload file without pre-existing GUID, consider to use \"./gen3-client upload\" instead.\n")

			logs.InitScoreBoard(0)
			filePaths, err := commonUtils.ParseFilePaths(filePath)
			if len(filePaths) > 1 {
				fmt.Println("More than 1 file location has been found. Do not use \"*\" in file path or provide a folder as file path.")
				return
			}
			if err != nil {
				log.Fatalln("File path parsing error: " + err.Error())
			}
			if len(filePaths) == 1 {
				filePath = filePaths[0]
			}
			if _, err := os.Stat(filePath); os.IsNotExist(err) {
				logs.AddToFailedLogMap(filePath, "", false)
				logs.WriteToFailedLog(false)
				logs.IncrementScore(len(logs.ScoreBoard) - 1)
				logs.PrintScoreBoard()
				log.Fatalf("The file you specified \"%s\" does not exist locally.", filePath)
			}

			file, err := os.Open(filePath)
			if err != nil {
				logs.AddToFailedLogMap(filePath, "", false)
				logs.WriteToFailedLog(false)
				logs.IncrementScore(len(logs.ScoreBoard) - 1)
				logs.PrintScoreBoard()
				log.Fatalln("File open error: " + err.Error())
			}
			defer file.Close()

			furObject := FileUploadRequestObject{FilePath: filePath, FileName: path.Base(filePath), GUID: guid}

			furObject, err = GenerateUploadRequest(furObject, file)
			if err != nil {
				file.Close()
				logs.AddToFailedLogMap(furObject.FilePath, furObject.PresignedURL, false)
				logs.WriteToFailedLog(false)
				logs.IncrementScore(len(logs.ScoreBoard) - 1)
				logs.PrintScoreBoard()
				log.Fatalf("Error occurred during request generation: %s", err.Error())
			}
			err = uploadFile(furObject)
			if err != nil {
				log.Println(err.Error())
				logs.IncrementScore(len(logs.ScoreBoard) - 1) // update failed score
			} else {
				logs.IncrementScore(0) // update succeeded score
			}
			logs.WriteToFailedLog(false)
			logs.PrintScoreBoard()
		},
	}

	uploadCmd.Flags().StringVar(&guid, "guid", "", "Specify the guid for the data you would like to work with")
	uploadCmd.MarkFlagRequired("guid")
	uploadCmd.Flags().StringVar(&filePath, "file", "", "Specify file to upload to with --file=~/path/to/file")
	uploadCmd.MarkFlagRequired("file")
	RootCmd.AddCommand(uploadCmd)
}
