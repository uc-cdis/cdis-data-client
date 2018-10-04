package g3cmd

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/uc-cdis/gen3-client/gen3-client/jwt"
)

/* performing function of download data */
func RequestDownload(resp *http.Response) *http.Response {
	/*
		Download file from given url encoded in resp
	*/

	if resp == nil {
		return nil
	}

	msg := jwt.JsonMessage{}

	str := jwt.ResponseToString(resp)
	if strings.Contains(str, "Can't find a location for the data") {
		log.Fatalf("The provided uuid is not found!!!")
	}

	jwt.DecodeJsonFromString(str, &msg)
	if msg.Url == "" {
		log.Fatalf("Can not get url from " + str)
	}

	presignedDownloadURL := msg.Url
	fmt.Println("Downloading data ...")

	respDown, err := http.Get(presignedDownloadURL)
	if err != nil {
		panic(err)
	}

	return respDown
}

func init() {
	var guid string
	var filePath string

	var downloadCmd = &cobra.Command{
		Use:   "download",
		Short: "download a file from a UUID",
		Long: `Gets a presigned URL for a file from a GUID and then downloads the specified file.
	Examples: ./gen3-client download --profile user1 --guid 206dfaa6-bcf1-4bc9-b2d0-77179f0f48fc --file=~/Documents/file_to_download.json 
	`,
		Run: func(cmd *cobra.Command, args []string) {

			request := new(jwt.Request)
			configure := new(jwt.Configure)
			function := new(jwt.Functions)

			function.Config = configure
			function.Request = request

			endPointPostfix := "/user/data/download/" + guid

			respDown := function.DoRequestWithSignedHeader(RequestDownload, profile, "", endPointPostfix)

			if respDown == nil {
				fmt.Printf("Download error: %s\n", respDown)
			} else {
				out, err := os.Create(filePath)
				if err != nil {
					log.Fatalf(err.Error())
				}
				defer out.Close()
				defer respDown.Body.Close()
				_, err = io.Copy(out, respDown.Body)
				if err != nil {
					panic(err)
				}

				fmt.Printf("Successfully downloaded %s to %s!\n", guid, filePath)
			}

		},
	}

	downloadCmd.Flags().StringVar(&guid, "guid", "", "Specify the guid for the data you would like to work with")
	downloadCmd.MarkFlagRequired("guid")
	downloadCmd.Flags().StringVar(&filePath, "file", "", "Specify file to download to with --file=~/path/to/file")
	downloadCmd.MarkFlagRequired("file")
	RootCmd.AddCommand(downloadCmd)
}
