package g3cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// putCmd represents the put command
var putCmd = &cobra.Command{
	Use:   "put",
	Short: "Send PUT HTTP Request to the gdcapi",
	Long: `Sends a PUT HTTP Request to upload files to the database.
Specify file type as json or tsv with --file_type (default json).
If no profile is specified, "default" profile is used for authentication.

Examples: ./gen3-client put --uri=/v0/submission/bpa/test --file=~/Documents/file_to_upload.json
	  ./gen3-client put --uri=/v0/submission/bpa/test --file=~/Documents/file_to_upload.tsv --file_type=tsv
	  ./gen3-client put --profile=user1 --uri=/v0/submission/bpa/test --file=~/Documents/file_to_upload.json
`,
	Run: func(cmd *cobra.Command, args []string) {

		fmt.Println("Use the command with upload option!!!")
		fmt.Println("./gen3-client upload --profile=user1 --uuid 206dfaa6-xxxx-xxxx-xxxx-77179f0f48fc --file=~/Documents/file_to_upload.json")
	},
}

func init() {
	RootCmd.AddCommand(putCmd)
}
