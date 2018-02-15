package cmd

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/spf13/cobra"
	"github.com/uc-cdis/cdis-data-client/jwt"
)

type GetRequest struct {
	Function jwt.FunctionInterface
	Request  jwt.RequestInterface
}

type GetRequestInterface interface {
	RequestGet(jwt.Credential, *url.URL, string) *http.Response
}

func (getRequest *GetRequest) RequestGet(cred jwt.Credential, host *url.URL, contentType string) *http.Response {
	uri = "/api/" + strings.TrimPrefix(uri, "/")

	// TODO: Replace here by function of JWT
	resp, err := getRequest.Function.SignedRequest("GET", host.Scheme+"://"+host.Host+"/user/data/download/"+uuid,
		nil, cred.AccessKey)

	if err != nil {
		panic(err)
	}

	if resp.StatusCode == 401 {
		//log.Fatalf("Access token is expired %d\n%s", resp.StatusCode, presignedDownloadUrl)
		client := &http.Client{}
		getRequest.Request.RequestNewAccessKey(client, cred.APIEndpoint+"/credentials/cdis/access_token", &cred)
		resp, err = getRequest.Function.SignedRequest("GET", host.Scheme+"://"+host.Host+"/user/data/download/"+uuid,
			nil, cred.AccessKey)
		if err != nil {
			panic(err)
		}
	}
	return resp
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
		utils := new(jwt.Utils)
		request := new(jwt.Request)
		request.Utils = utils
		configure := new(jwt.Configure)
		function := new(jwt.Functions)

		function.Utils = utils
		function.Config = configure
		function.Request = request

		getRequest := GetRequest{Function: function, Request: request}

		resp := function.DoRequestWithSignedHeader(getRequest.RequestGet, profile)
		fmt.Println(utils.ResponseToString(resp))
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
