package g3cmd

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/uc-cdis/gen3-client/gen3-client/jwt"
)

func uploadFile(guid string, filePath string, fileType string, signedURL string) {
	file, err := os.Open(filePath)
	if err != nil {
		log.Fatal("File Error")
	}
	defer file.Close()

	req, bar, err := GenerateUploadRequest(file, fileType, signedURL)
	if err != nil {
		log.Fatalf("Error occured during request generation: %s", err.Error())
		return
	}
	fmt.Println("Uploading data ...")
	bar.Start()

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		panic(err)
	}
	fmt.Println(jwt.ResponseToString(resp))
	fmt.Printf("Successfully uploaded file \"%s\" to GUID %s.\n", filePath, guid)
}

func init() {
	var guid string
	var filePath string
	var fileType string

	var uploadCmd = &cobra.Command{
		Use:     "upload",
		Short:   "Upload a file to a GUID",
		Long:    `Gets a presigned URL for which to upload a file associated with a GUID and then uploads the specified file.`,
		Example: `./gen3-client upload --profile user1 --guid f6923cf3-xxxx-xxxx-xxxx-14ab3f84f9d6 --file=~/Documents/file_to_upload`,
		Run: func(cmd *cobra.Command, args []string) {

			if _, err := os.Stat(filePath); os.IsNotExist(err) {
				log.Fatalf("The file you specified \"%s\" does not exist locally.", filePath)
			}

			request := new(jwt.Request)
			configure := new(jwt.Configure)
			function := new(jwt.Functions)

			function.Config = configure
			function.Request = request

			endPointPostfix := "/user/data/upload/" + guid

			signedURL, _, err := function.DoRequestWithSignedHeader(profile, "", endPointPostfix, nil)
			if err != nil && !strings.Contains(err.Error(), "No UUID found") {
				log.Fatalf("Upload error: %s!\n", err)
			} else {
				uploadFile(guid, filePath, fileType, signedURL)
			}
		},
	}

	uploadCmd.Flags().StringVar(&guid, "guid", "", "Specify the guid for the data you would like to work with")
	uploadCmd.MarkFlagRequired("guid")
	uploadCmd.Flags().StringVar(&filePath, "file", "", "Specify file to upload to with --file=~/path/to/file")
	uploadCmd.MarkFlagRequired("file")
	uploadCmd.Flags().StringVar(&fileType, "file-type", "json", "Specify file-type you're uploading with --file-type={json|tsv} (defaults to json)")
	RootCmd.AddCommand(uploadCmd)
}
