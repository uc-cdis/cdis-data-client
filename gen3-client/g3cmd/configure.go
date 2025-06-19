package g3cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/uc-cdis/gen3-client/gen3-client/commonUtils"
	"github.com/uc-cdis/gen3-client/gen3-client/jwt"
	"github.com/uc-cdis/gen3-client/gen3-client/logs"
)

var conf jwt.Configure // Why is this a global variable?

func init() {
	var credFile string
	var apiEndpoint string
	var useShepherd string
	var minShepherdVersion string
	var configureCmd = &cobra.Command{
		Use:   "configure",
		Short: "Add or modify a configuration profile to your config file",
		Long: `Configuration file located at ~/.gen3/gen3_client_config.ini
	If a field is left empty, the existing value (if it exists) will remain unchanged`,
		Example: `./gen3-client configure --profile=<profile-name> --cred=<path-to-credential/cred.json> --apiendpoint=https://data.mycommons.org`,
		Run: func(cmd *cobra.Command, args []string) {
			// don't initialize transmission logs for non-uploading related commands
			logs.SetToBoth()

			jwt.UpdateConfig(profile, apiEndpoint, credFile, useShepherd, minShepherdVersion)

		},
	}

	configureCmd.Flags().StringVar(&profile, "profile", "", "Specify profile to use")
	configureCmd.MarkFlagRequired("profile") //nolint:errcheck
	configureCmd.Flags().StringVar(&credFile, "cred", "", "Specify the credential file that you want to use")
	configureCmd.MarkFlagRequired("cred") //nolint:errcheck
	configureCmd.Flags().StringVar(&apiEndpoint, "apiendpoint", "", "Specify the API endpoint of the data commons")
	configureCmd.MarkFlagRequired("apiendpoint") //nolint:errcheck
	configureCmd.Flags().StringVar(&useShepherd, "use-shepherd", "", fmt.Sprintf("Enables or disables support for the Shepherd API. If enabled, gen3client will use the Shepherd API if available. (Default: %v)", commonUtils.DefaultUseShepherd))
	configureCmd.Flags().StringVar(&minShepherdVersion, "min-shepherd-version", "", fmt.Sprintf("Specify the minimum version of Shepherd that the gen3client will use if Shepherd is enabled. (Default: %v)", commonUtils.DefaultMinShepherdVersion))
	RootCmd.AddCommand(configureCmd)
}
