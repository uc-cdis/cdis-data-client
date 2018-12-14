package g3cmd

// Deprecated: Use upload instead.
import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/spf13/cobra"
	"github.com/uc-cdis/gen3-client/gen3-client/commonUtils"
	"github.com/uc-cdis/gen3-client/gen3-client/jwt"
	pb "gopkg.in/cheggaaa/pb.v1"
)

func uploadFile(req *http.Request, bar *pb.ProgressBar, guid string, filePath string) {
	fmt.Println("Uploading data ...")
	bar.Start()

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Fatalf("Error occured during upload: %s", err.Error())
		bar.Finish()
		return
	}
	bar.Finish()
	fmt.Println(jwt.ResponseToString(resp))
	fmt.Printf("Successfully uploaded file \"%s\" to GUID %s.\n", filePath, guid)
}

func init() {
	var guid string
	var filePath string

	var uploadCmd = &cobra.Command{
		Use:        "upload-old",
		Short:      "Upload a file to a GUID",
		Long:       `Gets a presigned URL for which to upload a file associated with a GUID and then uploads the specified file.`,
		Example:    `./gen3-client upload-old --profile=<profile-name> --guid=f6923cf3-xxxx-xxxx-xxxx-14ab3f84f9d6 --file=<path-to-file>,`,
		Deprecated: `use "./gen3-client upload" instead.`,
		Run: func(cmd *cobra.Command, args []string) {
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
				log.Fatalf("Error occured during request generation: %s", err.Error())
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
