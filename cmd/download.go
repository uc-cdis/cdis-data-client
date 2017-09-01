package cmd

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"

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
		access_key, secret_key, api_endpoint := parse_config(profile)
		if access_key == "" && secret_key == "" && api_endpoint == "" {
			return
		}

		out, err := os.Create(file_path)
		if err != nil {
			panic(err)
		}
		defer out.Close()

		host := strings.TrimPrefix(api_endpoint, "http://")
		host = strings.TrimPrefix(api_endpoint, "https://")

		// Get the presigned url first
		resp, err := gdcHmac.SignedGet("https://"+host+"/user/data/download/"+uuid, "userapi", access_key, secret_key)
                if err != nil {
                        panic(err)
                }
		defer resp.Body.Close()

                buf := new(bytes.Buffer)
                buf.ReadFrom(resp.Body)
                presigned_download_url := buf.String()
		if resp.StatusCode != 200 {
			log.Fatalf("Got response code %i\n%s",resp.StatusCode, presigned_download_url)
		}
		fmt.Println("Presigned URL to download: " + presigned_download_url)

		respDown, err := http.Get(presigned_download_url)
		if err != nil {
			panic(err)
		}
		defer respDown.Body.Close()
		_, err = io.Copy(out, respDown.Body)
		if err != nil  {
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
