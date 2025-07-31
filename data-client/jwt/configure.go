package jwt

//go:generate mockgen -destination=./data-client/mocks/mock_configure.go -package=mocks github.com/calypr/data-client/data-client/jwt ConfigureInterface

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/url"
	"os"
	"path"
	"regexp"
	"strings"

	"github.com/calypr/data-client/data-client/commonUtils"
	homedir "github.com/mitchellh/go-homedir"
	"gopkg.in/ini.v1"
)

var ErrProfileNotFound = errors.New("profile not found in config file")

type Credential struct {
	Profile            string
	KeyId              string
	APIKey             string
	AccessToken        string
	APIEndpoint        string
	UseShepherd        string
	MinShepherdVersion string
}

type Configure struct{}

type ConfigureInterface interface {
	ReadFile(string, string) string
	ValidateUrl(string) (*url.URL, error)
	GetConfigPath() (string, error)
	UpdateConfigFile(Credential) error
	ParseKeyValue(str string, expr string) (string, error)
	ParseConfig(profile string) (Credential, error)
}

func (conf *Configure) ReadFile(filePath string, fileType string) string {
	//Look in config file
	fullFilePath, err := commonUtils.GetAbsolutePath(filePath)
	if err != nil {
		log.Println("error occurred when parsing config file path: " + err.Error())
		return ""
	}
	if _, err := os.Stat(fullFilePath); err != nil {
		log.Println("File specified at " + fullFilePath + " not found")
		return ""
	}

	content, err := os.ReadFile(fullFilePath)
	if err != nil {
		log.Println("error occurred when reading file: " + err.Error())
		return ""
	}

	contentStr := string(content[:])

	if fileType == "json" {
		contentStr = strings.Replace(contentStr, "\n", "", -1)
	}
	return contentStr
}

func (conf *Configure) ValidateUrl(apiEndpoint string) (*url.URL, error) {
	parsedURL, err := url.Parse(apiEndpoint)
	if err != nil {
		return parsedURL, errors.New("Error occurred when parsing apiendpoint URL: " + err.Error())
	}
	if parsedURL.Host == "" {
		return parsedURL, errors.New("Invalid endpoint. A valid endpoint looks like: https://www.tests.com")
	}
	return parsedURL, nil
}

func (conf *Configure) ReadCredentials(filePath string) (*Credential, error) {
	var profileConfig Credential
	jsonContent := conf.ReadFile(filePath, "json")
	jsonContent = strings.Replace(jsonContent, "key_id", "KeyId", -1)
	jsonContent = strings.Replace(jsonContent, "api_key", "APIKey", -1)
	err := json.Unmarshal([]byte(jsonContent), &profileConfig)
	if err != nil {
		errs := fmt.Errorf("Cannot read json file: %s", err.Error())
		log.Println(errs.Error())
		return nil, errs
	}
	return &profileConfig, nil
}

func (conf *Configure) GetConfigPath() (string, error) {
	homeDir, err := homedir.Dir()
	if err != nil {
		return "", err
	}
	configPath := path.Join(homeDir + commonUtils.PathSeparator + ".gen3" + commonUtils.PathSeparator + "gen3_client_config.ini")
	return configPath, nil
}

func (conf *Configure) InitConfigFile() error {
	/*
		Make sure the config exists on start up
	*/
	configPath, err := conf.GetConfigPath()
	if err != nil {
		return err
	}

	if _, err := os.Stat(path.Dir(configPath)); os.IsNotExist(err) {
		osErr := os.Mkdir(path.Join(path.Dir(configPath)), os.FileMode(0777))
		if osErr != nil {
			return err
		}
		_, osErr = os.Create(configPath)
		if osErr != nil {
			return err
		}
	}
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		_, osErr := os.Create(configPath)
		if osErr != nil {
			return err
		}
	}
	_, err = ini.Load(configPath)

	return err
}

func (conf *Configure) UpdateConfigFile(profileConfig Credential) error {
	/*
		Overwrite the config file with new credential

		Args:
			profileConfig: Credential object represents config of a profile
			configPath: file path to config file
	*/
	configPath, err := conf.GetConfigPath()
	if err != nil {
		errs := fmt.Errorf("error occurred when getting config path: %s", err.Error())
		log.Println(errs.Error())
		return errs
	}
	cfg, err := ini.Load(configPath)
	if err != nil {
		errs := fmt.Errorf("error occurred when loading config file: %s", err.Error())
		log.Println(errs.Error())
		return errs
	}
	cfg.Section(profileConfig.Profile).Key("key_id").SetValue(profileConfig.KeyId)
	cfg.Section(profileConfig.Profile).Key("api_key").SetValue(profileConfig.APIKey)
	cfg.Section(profileConfig.Profile).Key("access_token").SetValue(profileConfig.AccessToken)
	cfg.Section(profileConfig.Profile).Key("api_endpoint").SetValue(profileConfig.APIEndpoint)
	cfg.Section(profileConfig.Profile).Key("use_shepherd").SetValue(profileConfig.UseShepherd)
	cfg.Section(profileConfig.Profile).Key("min_shepherd_version").SetValue(profileConfig.MinShepherdVersion)
	err = cfg.SaveTo(configPath)
	if err != nil {
		errs := fmt.Errorf("error occurred when saving config file: %s", err.Error())
		return errs
	}
	return nil
}

func (conf *Configure) ParseKeyValue(str string, expr string) (string, error) {
	r, err := regexp.Compile(expr)
	if err != nil {
		return "", fmt.Errorf("error occurred when parsing key/value: %v", err.Error())
	}
	match := r.FindStringSubmatch(str)
	if len(match) == 0 {
		return "", fmt.Errorf("No match found")
	}
	return match[1], nil
}

func (conf *Configure) ParseConfig(profile string) (Credential, error) {
	/*
		Looking profile in config file. The config file is a text file located at ~/.gen3 directory. It can
		contain more than 1 profile. If there is no profile found, the user is asked to run a command to
		create the profile

		The format of config file is described as following

		[profile1]
		key_id=key_id_example_1
		api_key=api_key_example_1
		access_token=access_token_example_1
		api_endpoint=http://localhost:8000
		use_shepherd=true
		min_shepherd_version=2.0.0

		[profile2]
		key_id=key_id_example_2
		api_key=api_key_example_2
		access_token=access_token_example_2
		api_endpoint=http://localhost:8000
		use_shepherd=false
		min_shepherd_version=

		Args:
			profile: the specific profile in config file
		Returns:
			An instance of Credential
	*/

	homeDir, err := homedir.Dir()
	if err != nil {
		errs := fmt.Errorf("Error occurred when getting home directory: %s", err.Error())
		return Credential{}, errs
	}
	configPath := path.Join(homeDir + commonUtils.PathSeparator + ".gen3" + commonUtils.PathSeparator + "gen3_client_config.ini")
	profileConfig := Credential{
		Profile:     profile,
		KeyId:       "",
		APIKey:      "",
		AccessToken: "",
		APIEndpoint: "",
	}
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return Credential{}, fmt.Errorf("%w Run configure command (with a profile if desired) to set up account credentials \n"+
			"Example: ./data-client configure --profile=<profile-name> --cred=<path-to-credential/cred.json> --apiendpoint=https://data.mycommons.org", ErrProfileNotFound)
	}

	// If profile not in config file, prompt user to set up config first
	cfg, err := ini.Load(configPath)
	if err != nil {
		errs := fmt.Errorf("Error occurred when reading config file: %s", err.Error())
		return Credential{}, errs
	}
	sec, err := cfg.GetSection(profile)
	if err != nil {
		return Credential{}, fmt.Errorf("%w: Need to run \"data-client configure --profile="+profile+" --cred=<path-to-credential/cred.json> --apiendpoint=<api_endpoint_url>\" first", ErrProfileNotFound)
	}
	// Read in API key, key ID and endpoint for given profile
	profileConfig.KeyId = sec.Key("key_id").String()
	if profileConfig.KeyId == "" {
		errs := fmt.Errorf("key_id not found in profile.")
		return Credential{}, errs
	}
	profileConfig.APIKey = sec.Key("api_key").String()
	if profileConfig.APIKey == "" {
		errs := fmt.Errorf("api_key not found in profile.")
		return Credential{}, errs
	}
	profileConfig.AccessToken = sec.Key("access_token").String()
	if profileConfig.AccessToken == "" {
		errs := fmt.Errorf("access_token not found in profile.")
		return Credential{}, errs
	}
	profileConfig.APIEndpoint = sec.Key("api_endpoint").String()
	if profileConfig.APIEndpoint == "" {
		errs := fmt.Errorf("api_endpoint not found in profile.")
		return Credential{}, errs
	}
	// UseShepherd and MinShepherdVersion are optional
	profileConfig.UseShepherd = sec.Key("use_shepherd").String()
	profileConfig.MinShepherdVersion = sec.Key("min_shepherd_version").String()
	return profileConfig, nil
}
