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
		access_key, secret_key, gdcapi_endpoint := parse_config(profile)
		if access_key == "" && secret_key == "" && gdcapi_endpoint == "" {
			return
		}
		client := &http.Client{}
		host := strings.TrimPrefix(gdcapi_endpoint, "http://")

		uri = strings.TrimPrefix(uri, "/")

		// Create and send request
		fmt.Println("file_type")
		fmt.Println(file_type)
		body := bytes.NewBufferString(read_file(file_path, file_type))
		req, err := http.NewRequest("PUT", "http://"+host+"/"+uri, body)
		if err != nil {
			panic(err)
		}
		req.Header.Add("Host", host)
		req.Header.Add("X-Amz-Date", time.Now().UTC().Format("20060102T150405Z"))
		if file_type == "json" {
			req.Header.Add("Content-Type", "application/json")
		} else {
			req.Header.Add("Content-Type", "text/tab-separated-values")
		}

		signed_req := gdcHmac.Sign(req, gdcHmac.Credentials{AccessKeyID: access_key, SecretAccessKey: secret_key}, "submission")

		// Display what came back
		resp, err := client.Do(signed_req)
		if err != nil {
			panic(err)
		}
		buf := new(bytes.Buffer)
		buf.ReadFrom(resp.Body)
		s := buf.String()
		fmt.Println(s)
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
