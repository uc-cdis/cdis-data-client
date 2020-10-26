package jwt

//go:generate mockgen -destination=mocks/mock_configure.go -package=mocks jwt ConfigureInterface

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/url"
	"os"
	"path"
	"regexp"
	"strings"

	homedir "github.com/mitchellh/go-homedir"
	"github.com/uc-cdis/gen3-client/gen3-client/commonUtils"
)

type Credential struct {
	KeyId              string
	APIKey             string
	AccessKey          string
	APIEndpoint        string
	UseShepherd        string
	MinShepherdVersion string
}

type Configure struct{}

type ConfigureInterface interface {
	ReadFile(string, string) string
	ValidateUrl(string) (*url.URL, error)
	ReadLines(Credential, []byte, string, string, string, string) ([]string, bool)
	UpdateConfigFile(Credential, []byte, string, string, string, string, string)
	ParseKeyValue(str string, expr string) (string, error)
	ParseConfig(profile string) Credential
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

	content, err := ioutil.ReadFile(fullFilePath)
	if err != nil {
		log.Println("error occurred when reading file: " + err.Error())
		return ""
	}

	contentStr := string(content[:])

	if fileType == "json" {
		contentStr = strings.Replace(contentStr, "\n", "", -1)
	}
	return string(content[:])
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

func (conf *Configure) ReadCredentials(filePath string) Credential {
	var configuration Credential
	jsonContent := conf.ReadFile(filePath, "json")
	jsonContent = strings.Replace(jsonContent, "key_id", "KeyId", -1)
	jsonContent = strings.Replace(jsonContent, "api_key", "APIKey", -1)
	err := json.Unmarshal([]byte(jsonContent), &configuration)
	if err != nil {
		log.Fatalln("Cannot read json file: " + err.Error())
	}
	return configuration
}

func (conf *Configure) TryReadConfigFile() (string, []byte, error) {
	/*
		Try to open config file. If not existed, create empty config file.
	*/
	homeDir, err := homedir.Dir()
	if err != nil {
		return "", nil, err
	}
	configPath := path.Join(homeDir + commonUtils.PathSeparator + ".gen3" + commonUtils.PathSeparator + "config")

	content, err := conf.TryReadFile(configPath)

	return configPath, content, err
}

func (conf *Configure) ReadLines(cred Credential, configContent []byte, apiEndpoint, useShepherd, minShepherdVersion, profile string) ([]string, bool) {
	/*
		Search profile in config file. Update new credential if found.
	*/
	lines := strings.Split(string(configContent), "\n")
	found := false
	for i := 0; i < len(lines); i += 6 {
		if lines[i] == "["+profile+"]" {
			if cred.KeyId != "" {
				lines[i+1] = "key_id=" + cred.KeyId
			}
			if cred.APIKey != "" {
				lines[i+2] = "api_key=" + cred.APIKey
			}
			lines[i+3] = "access_key=" + cred.AccessKey
			if apiEndpoint != "" {
				lines[i+4] = "api_endpoint=" + apiEndpoint
			}
			if useShepherd != "" {
				lines[i+5] = "use_shepherd=" + useShepherd
			}
			if minShepherdVersion != "" {
				lines[i+6] = "min_shepherd_version=" + minShepherdVersion
			}
			found = true
			break
		}
	}
	return lines, found
}

func (conf *Configure) UpdateConfigFile(cred Credential, configContent []byte, apiEndpoint, useShepherd, minShepherdVersion, configPath, profile string) {
	/*
		Overwrite the config file with new credential

		Args:
			cred: Credential
			configContent: config file content in byte format
			configPath: file path
			profile: profile name

	*/
	lines, found := conf.ReadLines(cred, configContent, apiEndpoint, useShepherd, minShepherdVersion, profile)
	if found {
		f, err := os.OpenFile(configPath, os.O_WRONLY|os.O_TRUNC, 0777)
		if err != nil {
			log.Fatalln("error occurred when opening config file: " + err.Error())
		}
		defer func() {
			if err := f.Close(); err != nil {
				log.Println("error occurred when closing config file: " + err.Error())
			}
		}()
		for i := 0; i < len(lines)-1; i++ {
			f.WriteString(lines[i] + "\n")
		}
	} else {
		f, err := os.OpenFile(configPath, os.O_APPEND|os.O_WRONLY, os.ModeAppend)
		if err != nil {
			log.Fatalln("error occurred when opening config file: " + err.Error())
		}
		defer func() {
			if err := f.Close(); err != nil {
				log.Println("error occurred when closing config file: " + err.Error())
			}
		}()

		_, err = f.WriteString("[" + profile + "]\n" +
			"key_id=" + cred.KeyId + "\n" +
			"api_key=" + cred.APIKey + "\n" +
			"access_key=" + cred.AccessKey + "\n" +
			"api_endpoint=" + apiEndpoint + "\n" +
			"use_shepherd=" + useShepherd + "\n" +
			"min_shepherd_version=" + minShepherdVersion + "\n\n")

		if err != nil {
			log.Println("error occurred when updating config: " + err.Error())
		}
	}
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

func (conf *Configure) ParseConfig(profile string) Credential {
	/*
		Looking profile in config file. The config file is a text file located at ~/.gen3 directory. It can
		contain more than 1 profile. If there is no profile found, the user is asked to run a command to
		create the profile

		The format of config file is described as following

		[profile1]
		key_id=key_id_example_1
		api_key=api_key_example_1
		access_key=access_key_example_1
		api_endpoint=http://localhost:8000
		use_shepherd=true
		min_shepherd_version=2.0.0

		[profile2]
		key_id=key_id_example_2
		api_key=api_key_example_2
		access_key=access_key_example_2
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
		log.Fatalln("Error occurred when getting home directory: " + err.Error())
	}
	configPath := path.Join(homeDir + commonUtils.PathSeparator + ".gen3" + commonUtils.PathSeparator + "config")
	cred := Credential{
		KeyId:       "",
		APIKey:      "",
		AccessKey:   "",
		APIEndpoint: "",
	}
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		log.Println("No config file found in ~/.gen3/")
		fmt.Println("Run configure command (with a profile if desired) to set up account credentials \n" +
			"Example: ./gen3-client configure --profile=<profile-name> --cred=<path-to-credential/cred.json> --apiendpoint=https://data.mycommons.org")
		return cred
	}

	// If profile not in config file, prompt user to set up config first
	content, err := ioutil.ReadFile(configPath)
	if err != nil {
		log.Fatalln("Error occurred when reading config file: " + err.Error())
	}
	lines := strings.Split(string(content), "\n")

	profileLine := -1
	for i := 0; i < len(lines); i++ {
		if lines[i] == "["+profile+"]" {
			profileLine = i
			break
		}
	}

	if profileLine == -1 {
		log.Fatalln("Profile not in config file. Need to run \"gen3-client configure --profile=" + profile + " --cred=<path-to-credential/cred.json> --apiendpoint=<api_endpoint_url>\" first")
	}
	// Read in access key, secret key, endpoint for given profile
	cred.KeyId, err = conf.ParseKeyValue(lines[profileLine+1], "^key_id=(\\S*)")
	if err != nil {
		log.Fatalf("key_id not found in profile. Err: %v", err)
	}
	cred.APIKey, err = conf.ParseKeyValue(lines[profileLine+2], "^api_key=(\\S*)")
	if err != nil {
		log.Fatalf("api_key not found in profile. Err: %v", err)
	}
	cred.AccessKey, err = conf.ParseKeyValue(lines[profileLine+3], "^access_key=(\\S*)")
	if err != nil {
		log.Fatalf("access_key not found in profile. Err: %v", err)
	}
	cred.APIEndpoint, err = conf.ParseKeyValue(lines[profileLine+4], "^api_endpoint=(\\S*)")
	if err != nil {
		log.Fatalf("api_endpoint not found in profile. Err: %v", err)
	}
	// UseShepherd and MinShepherdVersion are optional
	cred.UseShepherd, _ = conf.ParseKeyValue(lines[profileLine+5], "^use_shepherd=(\\S*)")
	cred.MinShepherdVersion, _ = conf.ParseKeyValue(lines[profileLine+6], "^min_shepherd_version=(\\S*)")
	return cred
}

func (conf *Configure) TryReadFile(filePath string) ([]byte, error) {
	if _, err := os.Stat(path.Dir(filePath)); os.IsNotExist(err) {
		os.Mkdir(path.Join(path.Dir(filePath)), os.FileMode(0777))
		os.Create(filePath)
	}
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		os.Create(filePath)
	}

	return ioutil.ReadFile(filePath)
}
