package cmd

import (
	"bytes"
	"fmt"
	"net/url"
	"strings"

	"github.com/spf13/cobra"
	"github.com/uc-cdis/cdis-data-client/gdcHmac"
)

var deleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Send DELETE HTTP Request for given URI",
	Long: `Deletes a given URI from the database. 
If no profile is specified, "default" profile is used for authentication. 

Examples: ./cdis-data-client delete --uri=v0/submission/bpa/test/entities/example_id
	  ./cdis-data-client delete --profile=user1 --uri=v0/submission/bpa/test/entities/1af1d0ab-efec-4049-98f0-ae0f4bb1bc64
`,
	Run: func(cmd *cobra.Command, args []string) {
		cred := ParseConfig(profile)
		if cred.APIKey == "" && cred.AccessKey == "" && cred.APIEndpoint == "" {
			return
		}
		content_type := "application/json"
		host, _ := url.Parse(cred.APIEndpoint)

		// Declared in ./root.go
		uri = "/api/" + strings.TrimPrefix(uri, "/")

		// Display what came back
		// TODO: Replace here by function of JWT
		resp, err := gdcHmac.SignedRequest("DELETE", host.Scheme+"://"+host.Host+uri,
			nil, content_type, "submission", cred.AccessKey, cred.APIKey)
		if err != nil {
			panic(err)
		}
		buf := new(bytes.Buffer)
		buf.ReadFrom(resp.Body)
		fmt.Println(buf.String())
	},
}

func init() {
	RootCmd.AddCommand(deleteCmd)
}
