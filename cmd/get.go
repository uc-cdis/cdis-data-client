package cmd

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"

	"github.com/spf13/cobra"
	"github.com/uc-cdis/cdis-data-client/jwt"
)

type GetRequest struct {
	Function jwt.FunctionInterface
	Request  jwt.RequestInterface
}

type GetRequestInterface interface {
	RequestGet(jwt.Credential, *url.URL, string) (*http.Response, error)
	GetDownloadPreSignedURL(jwt.Credential, *url.URL, string) (*http.Response, error)
}

func (getRequest *GetRequest) GetDownloadPreSignedURL(cred jwt.Credential, host *url.URL, contentType string) (*http.Response, error) {
	/*
		Get the presigned url for dwonload
	*/
	resp, err := getRequest.Function.SignedRequest("GET",
		host.Scheme+"://"+host.Host+"/user/data/download/"+uuid, nil, cred.AccessKey)
	defer resp.Body.Close()

	if err != nil {
		panic(err)
	}
	if resp.StatusCode != 200 {
		log.Fatalf("User error %d\n", resp.StatusCode)
	}

	return resp, err
}

func (getRequest *GetRequest) RequestGet(cred jwt.Credential, host *url.URL, contentType string) (*http.Response, error) {
	/*
		Download file from given url with
		Args:
			cred: crediential
			host: file address
			contentType: content type of the request
		Returns:
			httpResponse, error

	*/
	resp, err := getRequest.GetDownloadPreSignedURL(cred, host, contentType)
	message := JsonMessage{}
	err = json.Unmarshal([]byte(jwt.ResponseToString(resp)), &message)

	presignedDownloadURL := message.url
	fmt.Println("Downloading data from url: " + presignedDownloadURL)

	respDown, err := http.Get(presignedDownloadURL)
	return respDown, err
}

// getCmd represents the get command
var getCmd = &cobra.Command{
	Use:   "get",
	Short: "Send GET HTTP Request for given URI",
	Long: `Gets a given URI from the database.
If no profile is specified, "default" profile is used for authentication.

Examples: ./cdis-data-client get --uri=v0/submission/bpa/test/entities/example_id
	  ./cdis-data-client get --profile=user1 --uri=v0/submission/bpa/test/entities/1af1d0ab-efec-4049-98f0-ae0f4bb1bc64
`,
	Run: func(cmd *cobra.Command, args []string) {
		request := new(jwt.Request)
		configure := new(jwt.Configure)
		function := new(jwt.Functions)

		function.Config = configure
		function.Request = request

		getRequest := GetRequest{Function: function, Request: request}

		resp, _ := function.DoRequestWithSignedHeader(getRequest.RequestGet, profile, file_type)
		fmt.Println(jwt.ResponseToString(resp))
	},
}

func init() {
	RootCmd.AddCommand(getCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// getCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// getCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
