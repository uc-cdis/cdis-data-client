package g3cmd

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"jwt"
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
		log.Fatalf("The provided guid \"%s\" is not found.", uuid)
	}

	jwt.DecodeJsonFromString(str, &msg)
	if msg.Url == "" {
		log.Fatalf("Can not get url from " + str)
	}

	presignedUploadURL := msg.Url

	fmt.Println("Uploading data ...")
	// Create and send request
	data, err := ioutil.ReadFile(file_path)
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

var uploadCmd = &cobra.Command{
	Use:   "upload",
	Short: "Upload a file to a GUID",
	Long: `Gets a presigned URL for which to upload a file associated with a GUID and then uploads the specified file. 
Examples: ./gen3-client upload --profile user1 --guid f6923cf3-xxxx-xxxx-xxxx-14ab3f84f9d6 --file=~/Documents/file_to_upload
`,
	Run: func(cmd *cobra.Command, args []string) {
		if file_path == "" {
			log.Fatalf("Need to provide a file to upload using the --file option.")
		}

		if uuid == "" {
			log.Fatalf("Need to provide a guid to upload to using the --guid option.")
		}

		if _, err := os.Stat(file_path); os.IsNotExist(err) {
			log.Fatalf("The file you specified \"%s\" does not exist locally.", file_path)
		}

		request := new(jwt.Request)
		configure := new(jwt.Configure)
		function := new(jwt.Functions)

		function.Config = configure
		function.Request = request

		endPointPostfix := "/user/data/upload/" + uuid

		resp := function.DoRequestWithSignedHeader(RequestUpload, profile, "", endPointPostfix)
		if resp == nil {
			fmt.Println("Upload error: %s!", jwt.ResponseToString(resp))
		} else {
			fmt.Println(jwt.ResponseToString(resp))
			fmt.Println("Successfully uploaded file \"%s\" to GUID %s.", file_path, uuid)
		}
	},
}

func init() {
	RootCmd.AddCommand(uploadCmd)
}
