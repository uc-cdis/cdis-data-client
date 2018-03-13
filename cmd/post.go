package cmd

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
If no profile is specified, "default" profile is used for authentication. 

Examples: ./cdis-data-client put --uri=/v0/submission/graphql --file=~/Documents/my_grqphql_query.json
	  ./cdis-data-client put --profile=user1 --uri=/v0/submission/graphql --file=~/Documents/my_grqphql_query.json
`,
	Run: func(cmd *cobra.Command, args []string) {

		fmt.Println("Use the command with upload option!!!")
		fmt.Println("./cdis-data-client upload --profile=user1 --uuid 206dfaa6-xxxx-xxxx-xxxx-77179f0f48fc --file=~/Documents/file_to_upload.json")
	},
}

func init() {
	RootCmd.AddCommand(postCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// postCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// postCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
