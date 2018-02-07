package cmd

import (
	"bytes"
	"strings"

	"github.com/spf13/cobra"
	"github.com/uc-cdis/cdis-data-client/gdcHmac"
	"net/url"
	"net/http"
	"fmt"
)

func RequestPost(cred Credential, host *url.URL, contentType string) (*http.Response) {
	uri = "/api/" + strings.TrimPrefix(uri, "/")
	// Create and send request
	body := bytes.NewBufferString(ReadFile(file_path, file_type))

	if file_type == "tsv" {
		contentType = "text/tab-separated-values"
	}
	// TODO: Replace here by function of JWT
	resp, err := gdcHmac.SignedRequest("POST", host.Scheme+"://"+host.Host+uri,
		body, contentType, "submission", cred.AccessKey, cred.APIKey)

	if err != nil {
		panic(err)
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
		resp := DoRequestWithSignedHeader(RequestPost)
		fmt.Println(ResponseToString(resp))
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
