package g3cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// getCmd represents the get command
var getCmd = &cobra.Command{
	Use:   "get",
	Short: "Send GET HTTP Request for given URI",
	Long: `Gets a given URI from the database.
If no profile is specified, "default" profile is used for authentication.

Examples: ./gen3-client get --uri=/v0/submission/bpa/test/entities/example_id
	  ./gen3-client get --profile=user1 --guid 206dfaa6-bcf1-4bc9-b2d0-77179f0f48fc --file=~/Documents/file_to_download.json 
`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Use the command with download option!")
		fmt.Println("./gen3-client download --profile=user1 --guid 206dfaa6-xxxx-xxxx-xxxx-77179f0f48fc --file=~/Documents/file_to_download.json")
	},
}

func init() {
	RootCmd.AddCommand(getCmd)
}
