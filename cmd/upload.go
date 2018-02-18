package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"

	"net/url"

	"github.com/spf13/cobra"
	"github.com/uc-cdis/cdis-data-client/jwt"
)

type Upload struct {
	Function jwt.FunctionInterface
	Request  jwt.RequestInterface
}

type UploadInterface interface {
	RequestUpload(jwt.Credential, *url.URL, string) (*http.Response, error)
	GetUploadPreSignedURL(jwt.Credential, *url.URL, string) string
}

func (upload *Upload) GetUploadPreSignedURL(cred jwt.Credential, host *url.URL, contentType string) (*http.Response, error) {
	/*
	   Get presigned url for upload
	*/
	resp, err := upload.Function.SignedRequest("GET", host.Scheme+"://"+host.Host+"/user/data/upload/"+uuid,
		nil, cred.AccessKey)
	if err != nil {
		panic(err)
	}
	if resp.StatusCode != 200 {
		log.Fatalf("User error %d\n", resp.StatusCode)
	}
	return resp, err
}

func (upload *Upload) RequestUpload(cred jwt.Credential, host *url.URL, contentType string) (*http.Response, error) {
	/*
		Upload file with
		Args:
			cred: crediential
			host: file address
			contentType: content type of the request
		Returns:
			httpResponse, error

	*/
	resp, err := upload.GetUploadPreSignedURL(cred, host, contentType)
	message := JsonMessage{}
	err = json.Unmarshal([]byte(jwt.ResponseToString(resp)), &message)
	presignedUploadUrl := message.url

	fmt.Println("Uploading data to URL: " + presignedUploadUrl)
	// Create and send request
	data, err := ioutil.ReadFile(file_path)
	if err != nil {
		log.Fatal(err)
	}
	body := bytes.NewBufferString(string(data[:]))
	req, _ := http.NewRequest("PUT", presignedUploadUrl, body)

	client := &http.Client{}
	resp, err = client.Do(req)
	if err != nil {
		panic(err)
	}
	return resp, err
}

var uploadCmd = &cobra.Command{
	Use:   "upload",
	Short: "Upload a file to a UUID",
	Long: `Gets a presigned URL for which to upload a file associated with a UUID and then uploads the specified file. 
Examples: ./cdis-data-client upload --uuid --file=~/Documents/file_to_upload.json 
`,
	Run: func(cmd *cobra.Command, args []string) {
		request := new(jwt.Request)
		configure := new(jwt.Configure)
		function := new(jwt.Functions)

		function.Config = configure
		function.Request = request

		upload := Upload{Function: function, Request: request}
		respDown, _ := function.DoRequestWithSignedHeader(upload.RequestUpload, profile, file_type)
		fmt.Println(jwt.ResponseToString(respDown))
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
