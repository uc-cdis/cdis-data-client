package g3cmd

// Deprecated: Use upload instead.
import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/uc-cdis/gen3-client/gen3-client/logs"

	"github.com/spf13/cobra"
	"github.com/uc-cdis/gen3-client/gen3-client/commonUtils"
)

func uploadFile(furObject FileUploadRequestObject) {
	fmt.Println("Uploading data ...")
	furObject.Bar.Start()

	client := &http.Client{}
	_, err := client.Do(furObject.Request)
	if err != nil {
		log.Printf("Error occurred during upload: %s", err.Error())
		logs.AddToFailedLogMap(furObject.FilePath, furObject.PresignedURL, false)
		furObject.Bar.Finish()
		return
	}
	furObject.Bar.Finish()
	fmt.Printf("Successfully uploaded file \"%s\" to GUID %s.\n", furObject.FilePath, furObject.GUID)
	logs.DeleteFromFailedLogMap(furObject.FilePath, true)
	logs.WriteToSucceededLog(furObject.FilePath, furObject.GUID, false)
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

			filePaths, err := commonUtils.ParseFilePaths(filePath)
			if len(filePaths) > 1 {
				fmt.Println("More than 1 file location has been found. Do not use \"*\" in file path or provide a folder as file path.")
				return
			}
			if err != nil {
				panic(err)
			}
			if len(filePaths) == 1 {
				filePath = filePaths[0]
			}
			if _, err := os.Stat(filePath); os.IsNotExist(err) {
				log.Fatalf("The file you specified \"%s\" does not exist locally.", filePath)
			}

			file, err := os.Open(filePath)
			if err != nil {
				log.Fatalln("File open error: " + err.Error())
			}
			defer file.Close()

			furObject := FileUploadRequestObject{FilePath: filePath, GUID: guid}

			furObject, err = GenerateUploadRequest(furObject, file)
			if err != nil {
				file.Close()
				log.Fatalf("Error occurred during request generation: %s", err.Error())
			}
			uploadFile(furObject)
			logs.WriteToFailedLog(false)
		},
	}

	uploadCmd.Flags().StringVar(&guid, "guid", "", "Specify the guid for the data you would like to work with")
	uploadCmd.MarkFlagRequired("guid")
	uploadCmd.Flags().StringVar(&filePath, "file", "", "Specify file to upload to with --file=~/path/to/file")
	uploadCmd.MarkFlagRequired("file")
	RootCmd.AddCommand(uploadCmd)
}
