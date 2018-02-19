package cmd

import (
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/spf13/cobra"
	"github.com/uc-cdis/cdis-data-client/jwt"
)

func RequestGet(resp *http.Response) *http.Response {
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

// getCmd represents the get command
var getCmd = &cobra.Command{
	Use:   "get",
	Short: "Send GET HTTP Request for given URI",
	Long: `Gets a given URI from the database.
If no profile is specified, "default" profile is used for authentication.

Examples: ./cdis-data-client get --uri=/v0/submission/bpa/test/entities/example_id
	  ./cdis-data-client get --profile=user1 --uri=/v0/submission/bpa/test/entities/1af1d0ab-efec-4049-98f0-ae0f4bb1bc64
`,
	Run: func(cmd *cobra.Command, args []string) {
		request := new(jwt.Request)
		configure := new(jwt.Configure)
		function := new(jwt.Functions)

		function.Config = configure
		function.Request = request

		respDown := function.DoRequestWithSignedHeader(RequestGet, profile, file_type, uri)

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
	RootCmd.AddCommand(getCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// getCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// getCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
