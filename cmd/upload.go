package cmd

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strings"

	"github.com/spf13/cobra"
	"github.com/uc-cdis/cdis-data-client/gdcHmac"
)

var uploadCmd = &cobra.Command{
	Use:   "upload",
	Short: "Upload a file to a UUID",
	Long: `Gets a presigned URL for which to upload a file associated with a UUID and then uploads the specified file. 
Examples: ./cdis-data-client upload --uuid --file=~/Documents/file_to_upload.json 
`,
	Run: func(cmd *cobra.Command, args []string) {
		access_key, secret_key, api_endpoint := parse_config(profile)
		if access_key == "" && secret_key == "" && api_endpoint == "" {
			return
		}

		data, err := ioutil.ReadFile(file_path)
		if err != nil {
			log.Fatal(err)
		}
		body := bytes.NewBufferString(string(data[:]))
		
		client := &http.Client{}
		host := strings.TrimPrefix(api_endpoint, "http://")
		host = strings.TrimPrefix(api_endpoint, "https://")

		// Get the presigned url first
		resp, err := gdcHmac.SignedGet("https://"+host+"/user/data/upload/"+uuid, "userapi", access_key, secret_key)
                if err != nil {
                        panic(err)
                }
		defer resp.Body.Close()
                
		buf := new(bytes.Buffer)
                buf.ReadFrom(resp.Body)
                presigned_upload_url := buf.String()
		if resp.StatusCode != 200 {
                        log.Fatalf("Got response code %i\n%s",resp.StatusCode, presigned_upload_url)
                }
		fmt.Println("Presigned URL to upload: " + presigned_upload_url)

		// Create and send request
		req, err := http.NewRequest("PUT", presigned_upload_url, body)
		if err != nil {
			panic(err)
		}

		resp, err = client.Do(req)
		if err != nil {
			panic(err)
		}
		buf = new(bytes.Buffer)
		buf.ReadFrom(resp.Body)
		s := buf.String()
		fmt.Println(s)
	},
}

func init() {
	RootCmd.AddCommand(uploadCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// putCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// putCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
