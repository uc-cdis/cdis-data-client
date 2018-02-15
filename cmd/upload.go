package cmd

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"

	"net/url"

	"github.com/spf13/cobra"
	"github.com/uc-cdis/cdis-data-client/gdcHmac"
	"github.com/uc-cdis/cdis-data-client/jwt"
)

type Upload struct {
	Function jwt.FunctionInterface
	Request  jwt.RequestInterface
	Utils    jwt.UtilInterface
}

type UploadInterface interface {
	RequestUpload(jwt.Credential, *url.URL, string) *http.Response
}

func (upload *Upload) RequestUpload(cred jwt.Credential, host *url.URL, contentType string) *http.Response {
	// TODO: Replace here by function of JWT
	// Get the presigned url first
	resp, err := gdcHmac.SignedRequest("GET", host.Scheme+"://"+host.Host+"/user/data/upload/"+uuid,
		nil, contentType, "userapi", cred.AccessKey, cred.APIKey)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	presignedUploadUrl := upload.Utils.ResponseToString(resp)
	client := &http.Client{}
	if resp.StatusCode != 200 {
		fmt.Println("Got response code %d\n%s", resp.StatusCode, presignedUploadUrl)
		upload.Request.RequestNewAccessKey(client, cred.APIEndpoint+"/credentials/cdis/access_token", &cred)
		resp, err = upload.Function.SignedRequest("GET", host.Scheme+"://"+host.Host+"/user/data/download/"+uuid, nil, cred.AccessKey)
		presignedUploadUrl = upload.Utils.ResponseToString(resp)
	}
	fmt.Println("Uploading data from URL: " + presignedUploadUrl)

	// Create and send request
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
		utils := new(jwt.Utils)
		request := new(jwt.Request)
		request.Utils = utils
		configure := new(jwt.Configure)
		function := new(jwt.Functions)

		function.Utils = utils
		function.Config = configure
		function.Request = request

		upload := Upload{Function: function, Request: request, Utils: utils}

		respDown := function.DoRequestWithSignedHeader(upload.RequestUpload, profile)
		fmt.Println(utils.ResponseToString(respDown))
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
