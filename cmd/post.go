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

type PostRequest struct {
	Function  jwt.FunctionInterface
	Configure jwt.ConfigureInterface
	Request   jwt.RequestInterface
}

type PostRequestInterface interface {
	RequestPost(jwt.Credential, *url.URL, string) (*http.Response, error)
	GetUploadPreSignedURL(cred jwt.Credential, host *url.URL, contentType string) (*http.Response, error)
}

func (postRequest *PostRequest) GetUploadPreSignedURL(cred jwt.Credential, host *url.URL, contentType string) (*http.Response, error) {
	/*
	   Get presigned url for upload
	*/
	resp, err := postRequest.Function.SignedRequest("GET", host.Scheme+"://"+host.Host+"/user/data/upload/"+uuid,
		nil, cred.AccessKey)
	if err != nil {
		panic(err)
	}
	if resp.StatusCode != 200 {
		log.Fatalf("User error %d\n", resp.StatusCode)
	}
	return resp, err
}

func (postRequest *PostRequest) RequestPost(cred jwt.Credential, host *url.URL, contentType string) (*http.Response, error) {
	/*
		Upload file with
		Args:
			cred: crediential
			host: file address
			contentType: content type of the request
		Returns:
			httpResponse, error

	*/
	resp, err := postRequest.GetUploadPreSignedURL(cred, host, contentType)
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

// postCmd represents the post command
var postCmd = &cobra.Command{
	Use:   "post",
	Short: "Send POST HTTP Request to the gdcapi",
	Long: `Sends a POST HTTP Request to make graphql queries stored in 
local json files to the gdcapi. 
If no profile is specified, "default" profile is used for authentication. 

Examples: ./cdis-data-client put --uri=v0/submission/graphql --file=~/Documents/my_grqphql_query.json
	  ./cdis-data-client put --profile=user1 --uri=v0/submission/graphql --file=~/Documents/my_grqphql_query.json
`,
	Run: func(cmd *cobra.Command, args []string) {

		request := new(jwt.Request)
		configure := new(jwt.Configure)
		function := new(jwt.Functions)

		function.Config = configure
		function.Request = request

		postRequest := PostRequest{Function: function, Configure: configure, Request: request}

		resp, _ := function.DoRequestWithSignedHeader(postRequest.RequestPost, profile, file_type)
		fmt.Println(jwt.ResponseToString(resp))
	},
}

func init() {
	RootCmd.AddCommand(postCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// postCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// postCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
