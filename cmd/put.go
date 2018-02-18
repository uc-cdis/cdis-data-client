package cmd

import (
	"fmt"
	"net/http"
	"net/url"

	"github.com/spf13/cobra"
	"github.com/uc-cdis/cdis-data-client/jwt"
)

type PutRequest struct {
	Function  jwt.FunctionInterface
	Configure jwt.ConfigureInterface
	Request   jwt.RequestInterface
}

type PutRequestInterface interface {
	RequestPut(jwt.Credential, *url.URL, string) (*http.Response, error)
}

func (putRequest *PutRequest) RequestPut(cred jwt.Credential, host *url.URL, contentType string) (*http.Response, error) {
	// uri = "/api/" + strings.TrimPrefix(uri, "/")
	// // Create and send request
	// body := bytes.NewBufferString(putRequest.Configure.ReadFile(file_path, file_type))

	// if file_type == "tsv" {
	// 	contentType = "text/tab-separated-values"
	// }

	// // Display what came back
	// // TODO: Replace here by function of JWT
	// resp, err := gdcHmac.SignedRequest("PUT", host.Scheme+"://"+host.Host+uri, body,
	// 	contentType, "submission", cred.AccessKey, cred.APIKey)
	//return resp, err
	panic("Use upload instead !!!")
}

// putCmd represents the put command
var putCmd = &cobra.Command{
	Use:   "put",
	Short: "Send PUT HTTP Request to the gdcapi",
	Long: `Sends a PUT HTTP Request to upload files to the database.
Specify file type as json or tsv with --file_type (default json).
If no profile is specified, "default" profile is used for authentication.

Examples: ./cdis-data-client put --uri=v0/submission/bpa/test --file=~/Documents/file_to_upload.json
	  ./cdis-data-client put --uri=v0/submission/bpa/test --file=~/Documents/file_to_upload.tsv --file_type=tsv
	  ./cdis-data-client put --profile=user1 --uri=v0/submission/bpa/test --file=~/Documents/file_to_upload.json
`,
	Run: func(cmd *cobra.Command, args []string) {

		request := new(jwt.Request)
		configure := new(jwt.Configure)
		function := new(jwt.Functions)

		function.Config = configure
		function.Request = request

		putRequest := PutRequest{Function: function, Configure: configure, Request: request}

		resp, _ := function.DoRequestWithSignedHeader(putRequest.RequestPut, profile, file_type)
		fmt.Println(jwt.ResponseToString(resp))
	},
}

func init() {
	RootCmd.AddCommand(putCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// putCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// putCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
