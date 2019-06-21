package g3cmd

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/cavaliercoder/grab"
	"github.com/spf13/cobra"
	"github.com/uc-cdis/gen3-client/gen3-client/commonUtils"
	"github.com/uc-cdis/gen3-client/gen3-client/jwt"
)

func downloadFile(guid string, filePath string, signedURL string) {
	filePath = commonUtils.ParseRootPath(filePath)
	client := grab.NewClient()
	req, _ := grab.NewRequest(filePath, signedURL)

	if strings.Contains(signedURL, "X-Amz-Signature") {
		req.NoResume = true
	}
	// start download
	fmt.Printf("Downloading %v...\n", guid)
	resp := client.Do(req)

	// start UI loop
	t := time.NewTicker(500 * time.Millisecond)
	defer t.Stop()

Loop:
	for {
		select {
		case <-t.C:
			fmt.Printf("\033[1A  transferred %v / %v bytes (%.2f%%)\033[K\n",
				resp.BytesComplete(),
				resp.Size,
				100*resp.Progress())

		case <-resp.Done:
			// download is complete
			break Loop
		}
	}
	fmt.Printf("\033[1A\033[K")

	// check for errors
	if err := resp.Err(); err != nil {
		if resp != nil && resp.HTTPResponse != nil && resp.HTTPResponse.StatusCode >= 400 && resp.HTTPResponse.StatusCode < 500 {
			log.Printf("Download failed: %v\n", err)
			return
		}
		log.Fatalf("Fatal download failed: %v\n", err)
	}

	fmt.Printf("Successfully downloaded %v \n", resp.Filename)
}

func init() {
	var guid string
	var filePath string
	var protocol string

	var downloadCmd = &cobra.Command{
		Use:     "download-single",
		Short:   "Download a single file from a GUID",
		Long:    `Gets a presigned URL for a file from a GUID and then downloads the specified file.`,
		Example: `./gen3-client download-single --profile=<profile-name> --guid=206dfaa6-bcf1-4bc9-b2d0-77179f0f48fc --file=~/Documents/file_to_download.json`,
		Run: func(cmd *cobra.Command, args []string) {

			request := new(jwt.Request)
			configure := new(jwt.Configure)
			function := new(jwt.Functions)

			function.Config = configure
			function.Request = request

			protocolText := ""
			if protocol != "" {
				protocolText = "?protocol=" + protocol
			}

			endPointPostfix := "/user/data/download/" + guid + protocolText

			msg, err := function.DoRequestWithSignedHeader(profile, "", endPointPostfix, "", nil)

			if err != nil {
				log.Printf("Download error: %s\n", err)
			} else if msg.URL == "" {
				log.Printf("Error in getting download URL for object %s\n", guid)
			} else {
				downloadFile(guid, filePath, msg.URL)
			}
		},
	}

	downloadCmd.Flags().StringVar(&guid, "guid", "", "Specify the guid for the data you would like to work with")
	downloadCmd.MarkFlagRequired("guid")
	downloadCmd.Flags().StringVar(&filePath, "file", ".", "Specify file to download to with --file=~/path/to/file (default: .)")
	downloadCmd.Flags().StringVar(&protocol, "protocol", "", "Specify the preferred protocol with --protocol=gs (default: \"\")")
	RootCmd.AddCommand(downloadCmd)
}
