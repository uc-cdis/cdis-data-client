package cmd

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/uc-cdis/cdis-data-client/jwt"

	"net/url"

	"github.com/spf13/cobra"
)

type Download struct {
	function jwt.FunctionInterface
}

type DownloadInterface interface {
	RequestDownload(jwt.Credential, *url.URL, string) *http.Response
}

func (download *Download) RequestDownload(cred jwt.Credential, host *url.URL, contentType string) *http.Response {
	// Get the presigned url first
	resp, err := download.function.SignedRequest("GET", host.Scheme+"://"+host.Host+"/user/data/download/"+uuid, nil, cred.AccessKey)

	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	presignedDownloadUrl := jwt.ResponseToString(resp)
	if resp.StatusCode == 401 {
		log.Fatalf("Access token is expired %d\n%s", resp.StatusCode, presignedDownloadUrl)
		client := &http.Client{}
		download.function.RequestNewAccessKey(client, cred.APIEndpoint+"/credentials/cdis/access_token", &cred)
		resp, err = download.function.SignedRequest("GET", host.Scheme+"://"+host.Host+"/user/data/download/"+uuid, nil, cred.AccessKey)
	}

	fmt.Println("Downloading data from url: " + presignedDownloadUrl)

	respDown, err := http.Get(presignedDownloadUrl)
	if err != nil {
		panic(err)
	}
	return respDown
}

var downloadCmd = &cobra.Command{
	Use:   "download",
	Short: "download a file from a UUID",
	Long: `Gets a presigned URL for a file from a UUID and then downloads the specified file. 
Examples: ./cdis-data-client download --uuid --file=~/Documents/file_to_download.json 
`,
	Run: func(cmd *cobra.Command, args []string) {
		respDown := DoRequestWithSignedHeader(RequestDownload)

		out, err := os.Create(file_path)
		if err != nil {
			panic(err)
		}
		defer out.Close()

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
