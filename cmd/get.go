package cmd

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/uc-cdis/cdis-data-client/gdcHmac"
)

// getCmd represents the get command
var getCmd = &cobra.Command{
	Use:   "get",
	Short: "Send GET HTTP Request for given URI",
	Long: `Gets a given URI from the database. 
If no profile is specified, "default" profile is used for authentication. 

Examples: ./cdis-data-client get --uri=v0/submission/bpa/test/entities/example_id
	  ./cdis-data-client get --profile=user1 --uri=v0/submission/bpa/test/entities/1af1d0ab-efec-4049-98f0-ae0f4bb1bc64
`,
	Run: func(cmd *cobra.Command, args []string) {
		access_key, secret_key, gdcapi_endpoint := parse_config(profile)
		if access_key == "" && secret_key == "" && gdcapi_endpoint == "" {
			return
		}

		host := strings.TrimPrefix(gdcapi_endpoint, "http://")
		host = strings.TrimPrefix(host, "https://")
		uri = strings.TrimPrefix(uri, "/")

		resp, err := gdcHmac.SignedGet("https://"+host+"/"+uri, "submission", access_key, secret_key)
		if err != nil {
			panic(err)
		}
		buf := new(bytes.Buffer)
		buf.ReadFrom(resp.Body)
		s := buf.String()
		fmt.Println(s)
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
