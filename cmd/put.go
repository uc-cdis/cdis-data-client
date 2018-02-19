package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/uc-cdis/cdis-data-client/jwt"
)

// putCmd represents the put command
var putCmd = &cobra.Command{
	Use:   "put",
	Short: "Send PUT HTTP Request to the gdcapi",
	Long: `Sends a PUT HTTP Request to upload files to the database.
Specify file type as json or tsv with --file_type (default json).
If no profile is specified, "default" profile is used for authentication.

Examples: ./cdis-data-client put --uri=/v0/submission/bpa/test --file=~/Documents/file_to_upload.json
	  ./cdis-data-client put --uri=/v0/submission/bpa/test --file=~/Documents/file_to_upload.tsv --file_type=tsv
	  ./cdis-data-client put --profile=user1 --uri=/v0/submission/bpa/test --file=~/Documents/file_to_upload.json
`,
	Run: func(cmd *cobra.Command, args []string) {

		request := new(jwt.Request)
		configure := new(jwt.Configure)
		function := new(jwt.Functions)

		function.Config = configure
		function.Request = request

		fmt.Println(jwt.ResponseToString(
			function.DoRequestWithSignedHeader(RequestUpload, profile, file_type, uri)))
	},
}

func init() {
	RootCmd.AddCommand(putCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// putCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// putCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
