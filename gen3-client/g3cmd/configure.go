package g3cmd

import (
	"log"

	"github.com/spf13/cobra"
	"github.com/uc-cdis/gen3-client/gen3-client/jwt"
	"github.com/uc-cdis/gen3-client/gen3-client/logs"
)

var conf jwt.Configure

func init() {
	var credFile string
	var apiEndpoint string
	var configureCmd = &cobra.Command{
		Use:   "configure",
		Short: "Add or modify a configuration profile to your config file",
		Long: `Configuration file located at ~/.gen3/config
	If a field is left empty, the existing value (if it exists) will remain unchanged`,
		Example: `./gen3-client configure --profile=<profile-name> --cred=<path-to-credential/cred.json> --apiendpoint=https://data.mycommons.org`,
		Run: func(cmd *cobra.Command, args []string) {
			// don't initialize transmission logs for non-uploading related commands
			logs.SetToBoth()

			cred := conf.ReadCredentials(credFile)
			conf.ValidateUrl(apiEndpoint)

			// Store user info in ~/.gen3/config
			configPath, content, err := conf.TryReadConfigFile()
			if err != nil {
				log.Fatalln("Error occurred when trying to read config file: " + err.Error())
			}
			conf.UpdateConfigFile(cred, content, apiEndpoint, configPath, profile)
			logs.CloseMessageLog()
		},
	}

	configureCmd.Flags().StringVar(&profile, "profile", "", "Specify profile to use")
	configureCmd.MarkFlagRequired("profile")
	configureCmd.Flags().StringVar(&credFile, "cred", "", "Specify the credential file that you want to use")
	configureCmd.MarkFlagRequired("cred")
	configureCmd.Flags().StringVar(&apiEndpoint, "apiendpoint", "", "Specify the API endpoint of the data commons")
	configureCmd.MarkFlagRequired("apiendpoint")
	RootCmd.AddCommand(configureCmd)
}
