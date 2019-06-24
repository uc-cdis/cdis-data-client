package g3cmd

import (
	"log"
	"strings"

	"github.com/spf13/cobra"
	"github.com/uc-cdis/gen3-client/gen3-client/jwt"
)

func init() {
	var guid string
	var downloadPath string
	var protocol string
	var filenameFormat string

	var downloadCmd = &cobra.Command{
		Use:     "download-single",
		Short:   "Download a single file from a GUID",
		Long:    `Gets a presigned URL for a file from a GUID and then downloads the specified file.`,
		Example: `./gen3-client download-single --profile=<profile-name> --guid=206dfaa6-bcf1-4bc9-b2d0-77179f0f48fc`,
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

			guids := make([]string, 0)
			guids = append(guids, guid)
			downloadFile(guids, downloadPath, filenameFormat, protocol, 1)
		},
	}

	downloadCmd.Flags().StringVar(&guid, "guid", "", "Specify the guid for the data you would like to work with")
	downloadCmd.MarkFlagRequired("guid")
	downloadCmd.Flags().StringVar(&downloadPath, "download-path", ".", "The directory in which to store the downloaded files")
	downloadCmd.Flags().StringVar(&filenameFormat, "filename-format", "original", "format of filename to be used, including \"original\", \"guid\" and \"combined\" (default: original)")
	downloadCmd.Flags().StringVar(&protocol, "protocol", "", "Specify the preferred protocol with --protocol=gs (default: \"\")")
	RootCmd.AddCommand(downloadCmd)
}
