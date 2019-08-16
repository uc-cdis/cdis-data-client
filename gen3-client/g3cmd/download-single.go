package g3cmd

import (
	"strings"

	"github.com/spf13/cobra"
	"github.com/uc-cdis/gen3-client/gen3-client/commonUtils"
	"github.com/uc-cdis/gen3-client/gen3-client/jwt"
)

func init() {
	var guid string
	var downloadPath string
	var protocol string
	var filenameFormat string
	var overwrite bool
	var noPrompt bool

	var downloadSingleCmd = &cobra.Command{
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

			downloadPath = commonUtils.ParseRootPath(downloadPath)
			if !strings.HasSuffix(downloadPath, "/") {
				downloadPath += "/"
			}
			filenameFormat = strings.ToLower(strings.TrimSpace(filenameFormat))
			validateFilenameFormat(downloadPath, filenameFormat, overwrite, noPrompt)

			guids := make([]string, 0)
			guids = append(guids, guid)
			downloadFile(guids, downloadPath, filenameFormat, overwrite, protocol, 1)
		},
	}

	downloadSingleCmd.Flags().StringVar(&profile, "profile", "", "Specify profile to use")
	downloadSingleCmd.MarkFlagRequired("profile")
	downloadSingleCmd.Flags().StringVar(&guid, "guid", "", "Specify the guid for the data you would like to work with")
	downloadSingleCmd.MarkFlagRequired("guid")
	downloadSingleCmd.Flags().StringVar(&downloadPath, "download-path", ".", "The directory in which to store the downloaded files")
	downloadSingleCmd.Flags().StringVar(&filenameFormat, "filename-format", "original", "The format of filename to be used, including \"original\", \"guid\" and \"combined\"")
	downloadSingleCmd.Flags().BoolVar(&overwrite, "overwrite", false, "Only useful when \"--filename-format=original\", will overwrite any duplicates in \"download-path\" if set to true, will rename file by appending a counter value to its filename otherwise (default: false)")
	downloadSingleCmd.Flags().BoolVar(&noPrompt, "no-prompt", false, "If set to true, will not display user prompt message for confirmation (default: false)")
	downloadSingleCmd.Flags().StringVar(&protocol, "protocol", "", "Specify the preferred protocol with --protocol=gs (default: \"\")")
	RootCmd.AddCommand(downloadSingleCmd)
}
