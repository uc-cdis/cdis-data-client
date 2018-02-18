package cmd

import (
	"encoding/json"
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
	Function jwt.FunctionInterface
	Request  jwt.RequestInterface
}

type DownloadInterface interface {
	RequestDownload(jwt.Credential, *url.URL, string) (*http.Response, error)
	GetDownloadPreSignedURL(jwt.Credential, *url.URL, string) string
}

func (download *Download) GetDownloadPreSignedURL(cred jwt.Credential, host *url.URL, contentType string) (*http.Response, error) {
	/*
		Get the presigned url for dwonload
	*/
	resp, err := download.Function.SignedRequest("GET",
		host.Scheme+"://"+host.Host+"/user/data/download/"+uuid, nil, cred.AccessKey)
	defer resp.Body.Close()

	if err != nil {
		panic(err)
	}
	if resp.StatusCode != 200 {
		log.Fatalf("User error %d\n", resp.StatusCode)
	}

	return resp, err
}
func (download *Download) RequestDownload(cred jwt.Credential, host *url.URL, contentType string) (*http.Response, error) {
	/*
		Download file from given url with
		Args:
			cred: crediential
			host: file address
			contentType: content type of the request
		Returns:
			httpResponse, error

	*/
	resp, err := download.GetDownloadPreSignedURL(cred, host, contentType)
	message := JsonMessage{}
	err = json.Unmarshal([]byte(jwt.ResponseToString(resp)), &message)

	presignedDownloadURL := message.url
	fmt.Println("Downloading data from url: " + presignedDownloadURL)

	respDown, err := http.Get(presignedDownloadURL)
	return respDown, err
}

var downloadCmd = &cobra.Command{
	Use:   "download",
	Short: "download a file from a UUID",
	Long: `Gets a presigned URL for a file from a UUID and then downloads the specified file. 
Examples: ./cdis-data-client download --uuid --file=~/Documents/file_to_download.json 
`,
	Run: func(cmd *cobra.Command, args []string) {

		request := new(jwt.Request)
		configure := new(jwt.Configure)
		function := new(jwt.Functions)

		function.Config = configure
		function.Request = request

		download := Download{Function: function, Request: request}

		respDown, _ := function.DoRequestWithSignedHeader(download.RequestDownload, profile, file_type)

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
