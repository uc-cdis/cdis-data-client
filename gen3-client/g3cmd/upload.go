package g3cmd

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"

	"github.com/uc-cdis/gen3-client/gen3-client/jwt"

	"github.com/spf13/cobra"
)

func init() {
	var guid string
	var filePath string

	var uploadCmd = &cobra.Command{
		Use:   "upload",
		Short: "Upload a file to a GUID",
		Long: `Gets a presigned URL for which to upload a file associated with a GUID and then uploads the specified file. 
	Examples: ./gen3-client upload --profile user1 --guid f6923cf3-xxxx-xxxx-xxxx-14ab3f84f9d6 --file=~/Documents/file_to_upload
	`,
		Run: func(cmd *cobra.Command, args []string) {

			if _, err := os.Stat(filePath); os.IsNotExist(err) {
				log.Fatalf("The file you specified \"%s\" does not exist locally.", filePath)
			}

			request := new(jwt.Request)
			configure := new(jwt.Configure)
			function := new(jwt.Functions)

			function.Config = configure
			function.Request = request

			endPointPostfix := "/user/data/upload/" + guid

			respURL, err := function.DoRequestWithSignedHeader(profile, "", endPointPostfix)
			if err != nil {
				log.Fatalf("Upload error: %s!\n", err)
			} else {
				fmt.Println("Uploading data ...")
				// Create and send request
				data, err := ioutil.ReadFile(filePath)
				if err != nil {
					log.Fatal(err)
				}
				body := bytes.NewBufferString(string(data[:]))
				content_type := "application/json"
				if file_type == "tsv" {
					content_type = "text/tab-separated-values"
				}
				req, _ := http.NewRequest("PUT", respURL, body)
				req.Header.Set("content_type", content_type)
				client := &http.Client{}
				resp, err := client.Do(req)
				if err != nil {
					panic(err)
				}

				fmt.Println(jwt.ResponseToString(resp))
				fmt.Printf("Successfully uploaded file \"%s\" to GUID %s.\n", filePath, guid)
			}
		},
	}

	uploadCmd.Flags().StringVar(&guid, "guid", "", "Specify the guid for the data you would like to work with")
	uploadCmd.MarkFlagRequired("guid")
	uploadCmd.Flags().StringVar(&filePath, "file", "", "Specify file to upload to with --file=~/path/to/file")
	uploadCmd.MarkFlagRequired("file")
	RootCmd.AddCommand(uploadCmd)
}
