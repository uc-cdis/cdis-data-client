package g3cmd

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"

	"github.com/uc-cdis/gen3-client/gen3-client/jwt"

	"github.com/spf13/cobra"
)

func uploadFile(guid string, filePath string, fileType string, signedURL string) {
	fmt.Println("Uploading data ...")
	// Create and send request
	data, err := ioutil.ReadFile(filePath)
	if err != nil {
		log.Fatal(err)
	}
	body := bytes.NewBufferString(string(data[:]))
	contentType := "application/json"
	if fileType == "tsv" {
		contentType = "text/tab-separated-values"
	}
	req, _ := http.NewRequest(http.MethodPut, signedURL, body)
	req.Header.Set("content_type", contentType)

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

			signedURL, err := function.DoRequestWithSignedHeader(profile, "", endPointPostfix)
			if err != nil {
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
