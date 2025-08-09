package jwt

import (
	"fmt"
	"log"
	"strings"

	"github.com/calypr/data-client/data-client/commonUtils"
	"github.com/calypr/data-client/data-client/logs"
	"github.com/hashicorp/go-version"
)

func UpdateConfig(profile string, apiEndpoint string, credFile string, fenceToken string, useShepherd string, minShepherdVersion string) error {

	var conf Configure
	var req Request

	profileConfig, err := conf.ReadCredentials(credFile, fenceToken)
	if err != nil {
		return err
	}
	profileConfig.Profile = profile
	apiEndpoint = strings.TrimSpace(apiEndpoint)
	if apiEndpoint[len(apiEndpoint)-1:] == "/" {
		apiEndpoint = apiEndpoint[:len(apiEndpoint)-1]
	}
	parsedURL, err := conf.ValidateUrl(apiEndpoint)
	if err != nil {
		return fmt.Errorf("Errr occurred when validating apiendpoint URL: %s", err.Error())
	}

	prefixEndPoint := parsedURL.Scheme + "://" + parsedURL.Host
	if profileConfig.AccessToken == "" {
		err = req.RequestNewAccessToken(prefixEndPoint+commonUtils.FenceAccessTokenEndpoint, profileConfig)
		if err != nil {
			receivedErrorString := err.Error()
			errorMessageString := receivedErrorString
			if strings.Contains(receivedErrorString, "401") {
				errorMessageString = `Invalid credentials for apiendpoint '` + prefixEndPoint + `': check if your credentials are expired or incorrect`
			} else if strings.Contains(receivedErrorString, "404") || strings.Contains(receivedErrorString, "405") || strings.Contains(receivedErrorString, "no such host") {
				errorMessageString = `The provided apiendpoint '` + prefixEndPoint + `' is possibly not a valid Gen3 data commons`
			}
			return fmt.Errorf("Error occurred when validating profile config: %s", errorMessageString)
		}
	}
	profileConfig.APIEndpoint = apiEndpoint

	useShepherd = strings.TrimSpace(useShepherd)
	profileConfig.UseShepherd = useShepherd
	minShepherdVersion = strings.TrimSpace(minShepherdVersion)
	if minShepherdVersion != "" {
		_, err = version.NewVersion(minShepherdVersion)
		if err != nil {
			return fmt.Errorf("Error occurred when validating minShepherdVersion: %s", err.Error())
		}
	}
	profileConfig.MinShepherdVersion = minShepherdVersion

	// Store user info in ~/.gen3/gen3_client_config.ini
	err = conf.UpdateConfigFile(*profileConfig)
	if err != nil {
		return err
	}
	log.Println(`Profile '` + profile + `' has been configured successfully.`)
	err = logs.CloseMessageLog()
	if err != nil {
		log.Println(err.Error())
		return err
	}
	return nil

}
