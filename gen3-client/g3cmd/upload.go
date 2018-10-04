package g3cmd

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/uc-cdis/gen3-client/gen3-client/jwt"

	"github.com/spf13/cobra"
)

/* used to perform upload data */
func RequestUpload(resp *http.Response) *http.Response {
	/*
		Upload file with presigned url encoded in response's json
	*/

	if resp == nil {
		return nil
	}

	msg := jwt.JsonMessage{}
	str := jwt.ResponseToString(resp)
	if strings.Contains(str, "Can't find a location for the data") {
		log.Fatalf("The provided guid \"%s\" is not found.", guid)
	}

	jwt.DecodeJsonFromString(str, &msg)
	if msg.Url == "" {
		log.Fatalf("Can not get url from " + str)
	}

	presignedUploadURL := msg.Url

	fmt.Println("Uploading data ...")
	// Create and send request
	data, err := ioutil.ReadFile(filePath)
	if err != nil {
		log.Fatal(err)
	}
	body := bytes.NewBufferString(string(data[:]))
	content_type := "application/json"
	if file_type == "tsv" {
		content_type = "text/tab-separated-values"
	}
	req, _ := http.NewRequest("PUT", presignedUploadURL, body)
	req.Header.Set("content_type", content_type)
	client := &http.Client{}
	resp, err = client.Do(req)
	if err != nil {
		panic(err)
	}

	return resp
}

func init() {
	var guid string
	var filePath string

	var uploadCmd = &cobra.Command{
		Use:   "upload",
		Short: "Upload a file to a GUID",
		Long: `Gets a presigned URL for which to upload a file associated with a GUID and then uploads the specified file. 
	Examples: ./gen3-client upload --profile user1 --guid f6923cf3-xxxx-xxxx-xxxx-14ab3f84f9d6 --file=~/Documents/file_to_upload
	`,
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

			resp := function.DoRequestWithSignedHeader(RequestUpload, profile, "", endPointPostfix)
			if resp == nil {
				fmt.Printf("Upload error: %s!\n", jwt.ResponseToString(resp))
			} else {
				fmt.Println(jwt.ResponseToString(resp))
				fmt.Printf("Successfully uploaded file \"%s\" to GUID %s.\n", filePath, guid)
			}
		},
	}

	uploadCmd.Flags().StringVar(&guid, "guid", "", "Specify the guid for the data you would like to work with")
	uploadCmd.MarkFlagRequired("guid")
	uploadCmd.Flags().StringVar(&filePath, "file", "", "Specify file to upload to with --file=~/path/to/file")
	uploadCmd.MarkFlagRequired("file")
	RootCmd.AddCommand(uploadCmd)
}
