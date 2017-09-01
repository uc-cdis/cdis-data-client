package cmd

import (
	"bytes"
	"fmt"
	"net/http"
	"strings"
	"time"

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
		access_key, secret_key, api_endpoint := parse_config(profile)
		if access_key == "" && secret_key == "" && api_endpoint == "" {
			return
		}
		client := &http.Client{}
		host := strings.TrimPrefix(api_endpoint, "http://")

		// Declared in ./root.go
		uri = strings.TrimPrefix(uri, "/")

		// Create and send request
		req, _ := http.NewRequest("DELETE", "http://"+host+"/"+uri, nil)
		req.Header.Add("Host", host)
		req.Header.Add("X-Amz-Date", time.Now().UTC().Format("20060102T150405Z"))

		signed_req := gdcHmac.Sign(req, gdcHmac.Credentials{AccessKeyID: access_key, SecretAccessKey: secret_key}, "submission")

		// Display what came back
		resp, err := client.Do(signed_req)
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
