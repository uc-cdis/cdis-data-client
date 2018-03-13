package cmd

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

Examples: ./cdis-data-client get --uri=/v0/submission/bpa/test/entities/example_id
	  ./cdis-data-client get --profile=user1 --uuid 206dfaa6-bcf1-4bc9-b2d0-77179f0f48fc --file=~/Documents/file_to_download.json 
`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Use the command with download option!!!")
		fmt.Println("./cdis-data-client download --profile=user1 --uuid 206dfaa6-xxxx-xxxx-xxxx-77179f0f48fc --file=~/Documents/file_to_download.json")
	},
}

func init() {
	RootCmd.AddCommand(getCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// getCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// getCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
