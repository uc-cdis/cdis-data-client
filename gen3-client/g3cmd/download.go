package g3cmd

import (
	"log"
	"strings"

	"github.com/cavaliercoder/grab"
	"github.com/spf13/cobra"
	"github.com/uc-cdis/gen3-client/gen3-client/jwt"
)

func init() {
	var guid string
	var downloadPath string
	var protocol string
	var filenameFormat string

	var downloadCmd = &cobra.Command{
		Use:     "download",
		Short:   "Download a file from a GUID",
		Long:    `Gets a presigned URL for a file from a GUID and then downloads the specified file.`,
		Example: `./gen3-client download --profile=<profile-name> --guid=206dfaa6-bcf1-4bc9-b2d0-77179f0f48fc --file=~/Documents/file_to_download.json`,
		Run: func(cmd *cobra.Command, args []string) {

			request := new(jwt.Request)
			configure := new(jwt.Configure)
			function := new(jwt.Functions)

			function.Config = configure
			function.Request = request

			filenameFormat = strings.ToLower(strings.TrimSpace(filenameFormat))
			if filenameFormat != "original" && filenameFormat != "guid" && filenameFormat != "combined" {
				log.Fatalln("Invalid option found! Option \"filename-format\" can either be \"original\", \"guid\" or \"combined\" only")
			}

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
				reqs := make([]*grab.Request, 0)
				req, _ := grab.NewRequest(downloadPath, msg.URL)

				if strings.Contains(msg.URL, "X-Amz-Signature") {
					req.NoResume = true
				}
				reqs = append(reqs, req)
				downloadFile(1, reqs)
			}
		},
	}

	downloadCmd.Flags().StringVar(&guid, "guid", "", "Specify the guid for the data you would like to work with")
	downloadCmd.MarkFlagRequired("guid")
	downloadCmd.Flags().StringVar(&downloadPath, "download-path", ".", "The directory in which to store the downloaded files")
	downloadCmd.Flags().StringVar(&filenameFormat, "filename-format", "original", "format of filename to be used, including \"original\", \"guid\" and \"combined\"")
	downloadCmd.Flags().StringVar(&protocol, "protocol", "", "Specify the preferred protocol with --protocol=gs")
	RootCmd.AddCommand(downloadCmd)
}
