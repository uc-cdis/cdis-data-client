package g3cmd

import (
	"fmt"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/uc-cdis/gen3-client/gen3-client/commonUtils"
)

func init() {
	var path string
	var hiddenCmd = &cobra.Command{
		Use: "hidden",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println(commonUtils.IsHidden(filepath.Base(path)))
		},
	}
	hiddenCmd.Flags().StringVar(&path, "path", "", "path to location")
	hiddenCmd.MarkFlagRequired("path")
	RootCmd.AddCommand(hiddenCmd)
}
