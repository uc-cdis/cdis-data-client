package cmd

import (
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/spf13/cobra"

	"github.com/uc-cdis/cdis-data-client/jwt"
)

/* performing function of download data */
func RequestDownload(resp *http.Response) *http.Response {
	/*
		Download file from given url encoded in resp
	*/

	msg := jwt.JsonMessage{}
	jwt.DecodeJsonFromResponse(resp, &msg)

	presignedDownloadURL := msg.Url
	fmt.Println("Downloading data from url: " + presignedDownloadURL)

	respDown, err := http.Get(presignedDownloadURL)
	if err != nil {
		panic(err)
	}
	return respDown
}

// represent to download command
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

		endPointPostfix := "/user/data/download/" + uuid
		respDown := function.DoRequestWithSignedHeader(RequestDownload, profile, file_type, endPointPostfix)

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
