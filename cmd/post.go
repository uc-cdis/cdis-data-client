package cmd

import (
	"bytes"
	"strings"

	"fmt"
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
	RequestPost(jwt.Credential, *url.URL, string) *http.Response
}

func (postRequest *PostRequest) RequestPost(cred jwt.Credential, host *url.URL, contentType string) *http.Response {
	uri = "/api/" + strings.TrimPrefix(uri, "/")
	// Create and send request
	body := bytes.NewBufferString(postRequest.Configure.ReadFile(file_path, file_type))

	if file_type == "tsv" {
		contentType = "text/tab-separated-values"
	}

	resp, err := postRequest.Function.SignedRequest("POST", host.Scheme+"://"+host.Host+uri,
		body, cred.AccessKey)

	if err != nil {
		panic(err)
	}

	if resp.StatusCode == 401 {
		//log.Fatalf("Access token is expired %d\n%s", resp.StatusCode, presignedDownloadUrl)
		client := &http.Client{}
		postRequest.Request.RequestNewAccessKey(client, cred.APIEndpoint+"/credentials/cdis/access_token", &cred)
		resp, err = postRequest.Function.SignedRequest("POST", host.Scheme+"://"+host.Host+uri,
			body, cred.AccessKey)
		if err != nil {
			panic(err)
		}
	}

	return resp
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

		resp := function.DoRequestWithSignedHeader(postRequest.RequestPost, profile, file_type)
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
