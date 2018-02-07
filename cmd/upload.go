package cmd

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"

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
		cred := ParseConfig(profile)
		if cred.APIKey == "" && cred.AccessKey == "" && cred.APIEndpoint == "" {
			return
		}

		content_type := "application/json"
		host, _ := url.Parse(cred.APIEndpoint)

		data, err := ioutil.ReadFile(file_path)
		if err != nil {
			log.Fatal(err)
		}
		body := bytes.NewBufferString(string(data[:]))

		client := &http.Client{}

		// Get the presigned url first
		// TODO: Replace here by function of JWT
		resp, err := gdcHmac.SignedRequest("GET", host.Scheme+"://"+host.Host+"/user/data/upload/"+uuid,
			nil, content_type, "userapi", cred.AccessKey, cred.APIKey)
		if err != nil {
			panic(err)
		}
		defer resp.Body.Close()

		buf := new(bytes.Buffer)
		buf.ReadFrom(resp.Body)
		presigned_upload_url := buf.String()
		if resp.StatusCode != 200 {
			log.Fatalf("Got response code %d\n%s", resp.StatusCode, presigned_upload_url)
		}
		fmt.Println("Uploading data from URL: " + presigned_upload_url)

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
