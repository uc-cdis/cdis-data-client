package g3cmd

import (
	"github.com/spf13/cobra"
	"github.com/uc-cdis/gen3-client/gen3-client/jwt"
)

var conf jwt.Configure

func init() {
	var credFile string
	var configureCmd = &cobra.Command{
		Use:   "configure",
		Short: "Add or modify a configuration profile to your config file",
		Long: `Configuration file located at ~/.gen3/config
	Prompts for access_key, secret_key, and gdcapi endpoint
	If a field is left empty, the existing value (if it exists) will remain unchanged
	If no profile is specified, "default" profile is used`,
		Example: `./gen3-client configure --profile=user1 --cred cred.json`,
		Run: func(cmd *cobra.Command, args []string) {

			cred := conf.ReadCredentials(credFile)
			apiEndpoint := conf.ParseUrl()

			// Store user info in ~/.gen3/config
			configPath, content, err := conf.TryReadConfigFile()
			if err != nil {
				panic(err)
			}
			conf.UpdateConfigFile(cred, content, apiEndpoint, configPath, profile)
		},
	}
	configureCmd.Flags().StringVar(&credFile, "cred", "", "Specify the credential file that you want to use")
	configureCmd.MarkFlagRequired("cred")
	RootCmd.AddCommand(configureCmd)
}
