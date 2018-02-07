package cmd

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"

	"github.com/spf13/cobra"
	"github.com/uc-cdis/cdis-data-client/gdcHmac"
	"net/url"
)

func RequestUpload(cred Credential, host *url.URL, contentType string) (*http.Response) {
	// TODO: Replace here by function of JWT
	// Get the presigned url first
	resp, err := gdcHmac.SignedRequest("GET", host.Scheme+"://"+host.Host+"/user/data/upload/"+uuid,
		nil, contentType, "userapi", cred.AccessKey, cred.APIKey)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	presignedUploadUrl := ResponseToString(resp)
	if resp.StatusCode != 200 {
		log.Fatalf("Got response code %d\n%s", resp.StatusCode, presignedUploadUrl)
	}
	fmt.Println("Uploading data from URL: " + presignedUploadUrl)

	// Create and send request
	client := &http.Client{}
	data, err := ioutil.ReadFile(file_path)
	if err != nil {
		log.Fatal(err)
	}
	body := bytes.NewBufferString(string(data[:]))
	req, err := http.NewRequest("PUT", presignedUploadUrl, body)
	if err != nil {
		panic(err)
	}

	resp, err = client.Do(req)
	if err != nil {
		panic(err)
	}
	return resp
}

var uploadCmd = &cobra.Command{
	Use:   "upload",
	Short: "Upload a file to a UUID",
	Long: `Gets a presigned URL for which to upload a file associated with a UUID and then uploads the specified file. 
Examples: ./cdis-data-client upload --uuid --file=~/Documents/file_to_upload.json 
`,
	Run: func(cmd *cobra.Command, args []string) {
		resp := DoRequestWithSignedHeader(RequestUpload)
		fmt.Println(ResponseToString(resp))
	},
}

func init() {
	RootCmd.AddCommand(uploadCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// putCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// putCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
