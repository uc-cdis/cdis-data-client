package cmd

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/uc-cdis/cdis-data-client/jwt"
)

/* used to perform upload data */
func RequestUpload(resp *http.Response) *http.Response {
	/*
		Upload file with presigned url encoded in response's json
	*/

	msg := jwt.JsonMessage{}
	str := jwt.ResponseToString(resp)
	if strings.Contains(str, "Can't find a location for the data") {
		log.Fatalf("The provided uuid is not found!!!")
	}

	jwt.DecodeJsonFromString(str, &msg)
	if msg.Url == "" {
		log.Fatalf("Can not get url from " + str)
	}

	presignedUploadURL := msg.Url

	fmt.Println("Uploading data ...")
	// Create and send request
	data, err := ioutil.ReadFile(file_path)
	if err != nil {
		log.Fatal(err)
	}
	body := bytes.NewBufferString(string(data[:]))
	content_type := "application/json"
	if file_type == "tsv" {
		content_type = "text/tab-separated-values"
	}
	req, _ := http.NewRequest("PUT", presignedUploadURL, body)
	req.Header.Set("content_type", content_type)
	client := &http.Client{}
	resp, err = client.Do(req)
	if err != nil {
		panic(err)
	}

	return resp
}

/* represent to download command */
var uploadCmd = &cobra.Command{
	Use:   "upload",
	Short: "Upload a file to a UUID",
	Long: `Gets a presigned URL for which to upload a file associated with a UUID and then uploads the specified file. 
Examples: ./cdis-data-client upload --profile user1 --uuid f6923cf3-xxxx-xxxx-xxxx-14ab3f84f9d6 --file=~/Documents/file_to_upload
`,
	Run: func(cmd *cobra.Command, args []string) {
		if file_path == "" {
			log.Fatalf("Need to provide --file option !!!")
		}

		if uuid == "" {
			log.Fatalf("Need to provide --uuid option !!!")
		}

		if _, err := os.Stat(file_path); os.IsNotExist(err) {
			log.Fatalf("Uploading file is not existed !!!")
		}

		request := new(jwt.Request)
		configure := new(jwt.Configure)
		function := new(jwt.Functions)

		function.Config = configure
		function.Request = request

		endPointPostfix := "/user/data/upload/" + uuid

		fmt.Println(jwt.ResponseToString(
			function.DoRequestWithSignedHeader(RequestUpload, profile, "", endPointPostfix)))

		fmt.Println("Done!!!")
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
