package jwt

import (
	"log"
	"strings"

	"github.com/hashicorp/go-version"
	"github.com/uc-cdis/gen3-client/gen3-client/commonUtils"
	"github.com/uc-cdis/gen3-client/gen3-client/logs"
)

func UpdateConfig(profile string, apiEndpoint string, credFile string, useShepherd string, minShepherdVersion string) error {

	var conf Configure
	var req Request

	profileConfig := conf.ReadCredentials(credFile)
	profileConfig.Profile = profile
	apiEndpoint = strings.TrimSpace(apiEndpoint)
	if apiEndpoint[len(apiEndpoint)-1:] == "/" {
		apiEndpoint = apiEndpoint[:len(apiEndpoint)-1]
	}
	parsedURL, err := conf.ValidateUrl(apiEndpoint)
	if err != nil {
		log.Fatalln("Error occurred when validating apiendpoint URL: " + err.Error())
	}

	prefixEndPoint := parsedURL.Scheme + "://" + parsedURL.Host
	err = req.RequestNewAccessToken(prefixEndPoint+commonUtils.FenceAccessTokenEndpoint, &profileConfig)
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
	profileConfig.APIEndpoint = apiEndpoint

	useShepherd = strings.TrimSpace(useShepherd)
	profileConfig.UseShepherd = useShepherd
	minShepherdVersion = strings.TrimSpace(minShepherdVersion)
	if minShepherdVersion != "" {
		_, err = version.NewVersion(minShepherdVersion)
		if err != nil {
			log.Fatalln("Error occurred when validating minShepherdVersion: " + err.Error())
		}
	}
	profileConfig.MinShepherdVersion = minShepherdVersion

	// Store user info in ~/.gen3/gen3_client_config.ini
	conf.UpdateConfigFile(profileConfig)
	log.Println(`Profile '` + profile + `' has been configured successfully.`)
	err = logs.CloseMessageLog()
	if err != nil {
		log.Println(err.Error())
		return err
	}
	return nil

}
