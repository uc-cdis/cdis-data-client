package g3cmd

import (
	"github.com/spf13/cobra"
)

func init() {
	var generateTSVCmd = &cobra.Command{
		Use:        "generate-tsv",
		Short:      "Generate a file upload tsv from a template",
		Long:       `Fills in a Gen3 data file template with information from a directory of files.`,
		Deprecated: "please use an older version of data-client",
		Run:        func(cmd *cobra.Command, args []string) {},
	}

	RootCmd.AddCommand(generateTSVCmd)
}
