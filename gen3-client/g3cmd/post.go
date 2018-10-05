package g3cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// postCmd represents the post command
var postCmd = &cobra.Command{
	Use:   "post",
	Short: "Send POST HTTP Request to the gdcapi",
	Long: `Sends a POST HTTP Request to make graphql queries stored in 
local json files to the gdcapi. 
If no profile is specified, "default" profile is used for authentication.`,
	Example: `./gen3-client put --uri=/v0/submission/graphql --file=~/Documents/my_grqphql_query.json
	  ./gen3-client put --profile=user1 --uri=/v0/submission/graphql --file=~/Documents/my_grqphql_query.json`,
	Run: func(cmd *cobra.Command, args []string) {

		fmt.Println("Use the command with upload option!!!")
		fmt.Println("./gen3-client upload --profile=user1 --uuid 206dfaa6-xxxx-xxxx-xxxx-77179f0f48fc --file=~/Documents/file_to_upload.json")
	},
}

func init() {
	RootCmd.AddCommand(postCmd)
}
