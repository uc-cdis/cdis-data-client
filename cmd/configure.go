package cmd

import (
	"github.com/spf13/cobra"
	"github.com/uc-cdis/cdis-data-client/jwt"
)

var conf jwt.Configure
var credFile string

// configureCmd represents the command to configure profile
var configureCmd = &cobra.Command{
	Use:   "configure",
	Short: "Add or modify a configuration profile to your config file",
	Long: `Configuration file located at ~/.cdis/config
Prompts for access_key, secret_key, and gdcapi endpoint
If a field is left empty, the existing value (if it exists) will remain unchanged
If no profile is specified, "default" profile is used

Examples: ./cdis-data-client configure
	  ./cdis-data-client configure --profile=user1 --creds creds.json`,
	Run: func(cmd *cobra.Command, args []string) {
		// Prompt user for info

		cred := conf.ReadCredentials(credFile)
		apiEndpoint := conf.ParseUrl()

		// Store user info in ~/.cdis/config
		configPath, content, err := conf.TryReadConfigFile()
		if err != nil {
			panic(err)
		}
		conf.UpdateConfigFile(cred, content, apiEndpoint, configPath, "default")
	},
}

func init() {
	configureCmd.Flags().StringVar(&credFile, "cred", "", "Specify the credential file that you could like to use")
	RootCmd.AddCommand(configureCmd)
}
