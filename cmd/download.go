package cmd

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"

	"github.com/spf13/cobra"
	"github.com/uc-cdis/cdis-data-client/gdcHmac"
)

var downloadCmd = &cobra.Command{
	Use:   "download",
	Short: "download a file from a UUID",
	Long: `Gets a presigned URL for a file from a UUID and then downloads the specified file. 
Examples: ./cdis-data-client download --uuid --file=~/Documents/file_to_download.json 
`,
	Run: func(cmd *cobra.Command, args []string) {
		cred := ParseConfig(profile)
		if cred.APIKey == "" && cred.AccessKey == "" && cred.APIEndpoint == "" {
			return
		}

		content_type := "application/json"
		host, _ := url.Parse(cred.APIEndpoint)

		out, err := os.Create(file_path)
		if err != nil {
			panic(err)
		}
		defer out.Close()

		// Get the presigned url first
		// TODO: Replace here by function of JWT
		resp, err := gdcHmac.SignedRequest(
			"GET", host.Scheme+"://"+host.Host+"/user/data/download/"+uuid, nil, content_type,
			"userapi", cred.AccessKey, cred.APIKey)
		if err != nil {
			panic(err)
		}
		defer resp.Body.Close()

		buf := new(bytes.Buffer)
		buf.ReadFrom(resp.Body)
		presigned_download_url := buf.String()
		if resp.StatusCode != 200 {
			log.Fatalf("Got response code %d\n%s", resp.StatusCode, presigned_download_url)
		}
		fmt.Println("Downloading data from url: " + presigned_download_url)

		respDown, err := http.Get(presigned_download_url)
		if err != nil {
			panic(err)
		}
		defer respDown.Body.Close()
		_, err = io.Copy(out, respDown.Body)
		if err != nil {
			panic(err)
		}
	},
}

func init() {
	RootCmd.AddCommand(downloadCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// putCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// putCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
