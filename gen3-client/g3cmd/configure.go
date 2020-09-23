package g3cmd

import (
	"fmt"
	"log"
	"strings"

	"github.com/hashicorp/go-version"
	"github.com/spf13/cobra"
	"github.com/uc-cdis/gen3-client/gen3-client/commonUtils"
	"github.com/uc-cdis/gen3-client/gen3-client/jwt"
	"github.com/uc-cdis/gen3-client/gen3-client/logs"
)

var conf jwt.Configure
var req jwt.Request

func init() {
	var credFile string
	var apiEndpoint string
	var useShepherd string
	var minShepherdVersion string
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
			apiEndpoint = strings.TrimSpace(apiEndpoint)
			if apiEndpoint[len(apiEndpoint)-1:] == "/" {
				apiEndpoint = apiEndpoint[:len(apiEndpoint)-1]
			}
			parsedURL, err := conf.ValidateUrl(apiEndpoint)
			if err != nil {
				log.Fatalln("Error occurred when validating apiendpoint URL: " + err.Error())
			}

			prefixEndPoint := parsedURL.Scheme + "://" + parsedURL.Host
			err = req.RequestNewAccessKey(prefixEndPoint+commonUtils.FenceAccessTokenEndpoint, &cred)
			if err != nil {
				receivedErrorString := err.Error()
				errorMessageString := receivedErrorString
				if strings.Contains(receivedErrorString, "401") {
					errorMessageString = `Invalid credentials for apiendpoint '` + prefixEndPoint + `': check if your credentials are expired or incorrect`
				} else if strings.Contains(receivedErrorString, "404") || strings.Contains(receivedErrorString, "405") || strings.Contains(receivedErrorString, "no such host") {
					errorMessageString = `The provided apiendpoint '` + prefixEndPoint + `' is possibly not a valid Gen3 data commons`
				}
				log.Fatalln("Error occurred when validating profile config: " + errorMessageString)
			}

			useShepherd = strings.TrimSpace(useShepherd)
			minShepherdVersion = strings.TrimSpace(minShepherdVersion)
			if minShepherdVersion != "" {
				_, err = version.NewVersion(minShepherdVersion)
				if err != nil {
					log.Fatalln("Error occurred when validating minShepherdVersion: " + err.Error())
				}
			}

			// Store user info in ~/.gen3/config
			configPath, content, err := conf.TryReadConfigFile()
			if err != nil {
				log.Fatalln("Error occurred when trying to read config file: " + err.Error())
			}
			conf.UpdateConfigFile(cred, content, apiEndpoint, useShepherd, minShepherdVersion, configPath, profile)
			log.Println(`Profile '` + profile + `' has been configured successfully.`)
			logs.CloseMessageLog()
		},
	}

	configureCmd.Flags().StringVar(&profile, "profile", "", "Specify profile to use")
	configureCmd.MarkFlagRequired("profile")
	configureCmd.Flags().StringVar(&credFile, "cred", "", "Specify the credential file that you want to use")
	configureCmd.MarkFlagRequired("cred")
	configureCmd.Flags().StringVar(&apiEndpoint, "apiendpoint", "", "Specify the API endpoint of the data commons")
	configureCmd.MarkFlagRequired("apiendpoint")
	configureCmd.Flags().StringVar(&useShepherd, "use-shepherd", "", fmt.Sprintf("Enables or disables support for the Shepherd API. If enabled, gen3client will use the Shepherd API if available. (Default: %v)", commonUtils.DefaultUseShepherd))
	configureCmd.Flags().StringVar(&minShepherdVersion, "min-shepherd-version", "", fmt.Sprintf("Specify the minimum version of Shepherd that the gen3client will use if Shepherd is enabled. (Default: %v)", commonUtils.DefaultMinShepherdVersion))
	RootCmd.AddCommand(configureCmd)
}
