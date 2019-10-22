package g3cmd

import (
	"github.com/spf13/cobra"
	"github.com/uc-cdis/gen3-client/gen3-client/logs"
)

func init() {
	var guid string
	var downloadPath string
	var protocol string
	var filenameFormat string
	var rename bool
	var noPrompt bool
	var skipCompleted bool

	var downloadSingleCmd = &cobra.Command{
		Use:     "download-single",
		Short:   "Download a single file from a GUID",
		Long:    `Gets a presigned URL for a file from a GUID and then downloads the specified file.`,
		Example: `./gen3-client download-single --profile=<profile-name> --guid=206dfaa6-bcf1-4bc9-b2d0-77179f0f48fc`,
		Run: func(cmd *cobra.Command, args []string) {
			// don't initialize transmission logs for non-uploading related commands
			logs.SetToBoth()

			guids := make([]string, 0)
			guids = append(guids, guid)
			downloadFile(guids, downloadPath, filenameFormat, rename, noPrompt, protocol, 1, skipCompleted)
			logs.CloseMessageLog()
		},
	}

	downloadSingleCmd.Flags().StringVar(&profile, "profile", "", "Specify profile to use")
	downloadSingleCmd.MarkFlagRequired("profile")
	downloadSingleCmd.Flags().StringVar(&guid, "guid", "", "Specify the guid for the data you would like to work with")
	downloadSingleCmd.MarkFlagRequired("guid")
	downloadSingleCmd.Flags().StringVar(&downloadPath, "download-path", ".", "The directory in which to store the downloaded files")
	downloadSingleCmd.Flags().StringVar(&filenameFormat, "filename-format", "original", "The format of filename to be used, including \"original\", \"guid\" and \"combined\"")
	downloadSingleCmd.Flags().BoolVar(&rename, "rename", false, "Only useful when \"--filename-format=original\", will rename file by appending a counter value to its filename if set to true, otherwise the same filename will be used")
	downloadSingleCmd.Flags().BoolVar(&noPrompt, "no-prompt", false, "If set to true, will not display user prompt message for confirmation")
	downloadSingleCmd.Flags().StringVar(&protocol, "protocol", "", "Specify the preferred protocol with --protocol=gs")
	downloadSingleCmd.Flags().BoolVar(&skipCompleted, "skip-completed", false, "If set to true, will check for filename and size before download and skip any files in \"download-path\" that matches both")
	RootCmd.AddCommand(downloadSingleCmd)
}
