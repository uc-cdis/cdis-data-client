package g3cmd

// Deprecated: Use upload instead.
import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/spf13/cobra"
	"github.com/uc-cdis/gen3-client/gen3-client/commonUtils"
	pb "gopkg.in/cheggaaa/pb.v1"
)

func uploadFile(req *http.Request, bar *pb.ProgressBar, guid string, filePath string) {
	fmt.Println("Uploading data ...")
	bar.Start()

	client := &http.Client{}
	_, err := client.Do(req)
	if err != nil {
		log.Fatalf("Error occurred during upload: %s", err.Error())
		bar.Finish()
		return
	}
	bar.Finish()
	fmt.Printf("Successfully uploaded file \"%s\" to GUID %s.\n", filePath, guid)
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
				log.Fatal("File open error")
			}
			defer file.Close()

			req, bar, err := GenerateUploadRequest(guid, "", file)
			if err != nil {
				log.Fatalf("Error occurred during request generation: %s", err.Error())
				return
			}
			uploadFile(req, bar, guid, filePath)
		},
	}

	uploadCmd.Flags().StringVar(&guid, "guid", "", "Specify the guid for the data you would like to work with")
	uploadCmd.MarkFlagRequired("guid")
	uploadCmd.Flags().StringVar(&filePath, "file", "", "Specify file to upload to with --file=~/path/to/file")
	uploadCmd.MarkFlagRequired("file")
	RootCmd.AddCommand(uploadCmd)
}
