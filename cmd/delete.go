package cmd

import (
	"strings"

	"fmt"
	"net/http"
	"net/url"

	"github.com/spf13/cobra"
	"github.com/uc-cdis/cdis-data-client/gdcHmac"

	"github.com/uc-cdis/cdis-data-client/jwt"
)

type DeleteRequest struct {
	Function jwt.FunctionInterface
	Utils    jwt.UtilInterface
}

type DeleteRequestInterface interface {
	RequestDelete(jwt.Credential, *url.URL, string) *http.Response
}

func (delRequest *DeleteRequest) RequestDelete(cred jwt.Credential, host *url.URL, contentType string) *http.Response {
	// Declared in ./root.go
	uri = "/api/" + strings.TrimPrefix(uri, "/")

	// Display what came back
	// TODO: Replace here by function of JWT
	resp, err := gdcHmac.SignedRequest("DELETE", host.Scheme+"://"+host.Host+uri,
		nil, contentType, "submission", cred.AccessKey, cred.APIKey)
	if err != nil {
		panic(err)
	}
	return resp
}

var deleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Send DELETE HTTP Request for given URI",
	Long: `Deletes a given URI from the database. 
If no profile is specified, "default" profile is used for authentication. 

Examples: ./cdis-data-client delete --uri=v0/submission/bpa/test/entities/example_id
	  ./cdis-data-client delete --profile=user1 --uri=v0/submission/bpa/test/entities/1af1d0ab-efec-4049-98f0-ae0f4bb1bc64
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

		delRequest := DeleteRequest{Function: function, Utils: utils}

		resp := function.DoRequestWithSignedHeader(delRequest.RequestDelete, profile)
		fmt.Println(utils.ResponseToString(resp))
	},
}

func init() {
	RootCmd.AddCommand(deleteCmd)
}
