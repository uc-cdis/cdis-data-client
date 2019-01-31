package jwt

//go:generate mockgen -destination=mocks/mock_configure.go -package=mocks jwt ConfigureInterface

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/url"
	"os"
	"os/user"
	"path"
	"regexp"
	"strings"

	"github.com/uc-cdis/gen3-client/gen3-client/commonUtils"
)

type Credential struct {
	KeyId       string
	APIKey      string
	AccessKey   string
	APIEndpoint string
}

type Configure struct{}

type ConfigureInterface interface {
	ReadFile(string, string) string
	ValidateUrl(string)
	ReadLines(Credential, []byte, string, string) ([]string, bool)
	UpdateConfigFile(Credential, []byte, string, string, string)
	ParseKeyValue(str string, expr string, errMsg string) string
	ParseConfig(profile string) Credential
}

func (conf *Configure) ReadFile(filePath string, fileType string) string {
	//Look in config file
	fullFilePaths, err := commonUtils.ParseFilePaths(filePath)
	if len(fullFilePaths) > 1 {
		fmt.Println("More than 1 file location has been found. Do not use \"*\" in file path or provide a folder as file path.")
		return ""
	}
	if err != nil {
		panic(err)
	}
	var fullFilePath = filePath
	if len(fullFilePaths) == 1 {
		fullFilePath = fullFilePaths[0]
	}
	if _, err := os.Stat(fullFilePath); err != nil {
		fmt.Println("File specified at " + fullFilePath + " not found")
		return ""
	}

	content, err := ioutil.ReadFile(fullFilePath)
	if err != nil {
		panic(err)
	}

	contentStr := string(content[:])

	if fileType == "json" {
		contentStr = strings.Replace(contentStr, "\n", "", -1)
	}
	return string(content[:])
}

func (conf *Configure) ValidateUrl(apiEndpoint string) {
	parsedUrl, err := url.Parse(apiEndpoint)
	if err != nil {
		panic(err)
	}
	if parsedUrl.Host == "" {
		fmt.Print("Invalid endpoint. A valid endpoint looks like: https://www.tests.com\n")
		os.Exit(1)
	}
}

func (conf *Configure) ReadCredentials(filePath string) Credential {
	var configuration Credential
	jsonContent := conf.ReadFile(filePath, "json")
	jsonContent = strings.Replace(jsonContent, "key_id", "KeyId", -1)
	jsonContent = strings.Replace(jsonContent, "api_key", "APIKey", -1)
	err := json.Unmarshal([]byte(jsonContent), &configuration)
	if err != nil {
		fmt.Println("Cannot read json file: " + err.Error())
		os.Exit(1)
	}
	return configuration
}

func (conf *Configure) TryReadConfigFile() (string, []byte, error) {
	/*
		Try to open config file. If not existed, create empty config file.
	*/
	usr, err := user.Current()
	homeDir := ""
	if err == nil {
		homeDir = usr.HomeDir
	}
	configPath := path.Join(homeDir + "/.gen3/config")

	content, err := conf.TryReadFile(configPath)

	return configPath, content, err
}

func (conf *Configure) ReadLines(cred Credential, configContent []byte, apiEndpoint string, profile string) ([]string, bool) {
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
			found = true
			break
		}
	}
	return lines, found
}

func (conf *Configure) UpdateConfigFile(cred Credential, configContent []byte, apiEndpoint string, configPath string, profile string) {
	/*
		Overwrite the config file with new credential

		Args:
			cred: Credential
			configContent: config file content in byte format
			configPath: file path
			profile: profile name

	*/
	apiEndpoint = strings.TrimSpace(apiEndpoint)
	if apiEndpoint[len(apiEndpoint)-1:] == "/" {
		apiEndpoint = apiEndpoint[:len(apiEndpoint)-1]
	}

	lines, found := conf.ReadLines(cred, configContent, apiEndpoint, profile)
	if found {
		f, err := os.OpenFile(configPath, os.O_WRONLY|os.O_TRUNC, 0777)
		if err != nil {
			panic(err)
		}
		defer func() {
			if err := f.Close(); err != nil {
				panic(err)
			}
		}()
		for i := 0; i < len(lines)-1; i++ {
			f.WriteString(lines[i] + "\n")
		}
	} else {
		f, err := os.OpenFile(configPath, os.O_APPEND|os.O_WRONLY, os.ModeAppend)
		if err != nil {
			panic(err)
		}
		defer func() {
			if err := f.Close(); err != nil {
				panic(err)
			}
		}()

		_, err = f.WriteString("[" + profile + "]\n" +
			"key_id=" + cred.KeyId + "\n" +
			"api_key=" + cred.APIKey + "\n" +
			"access_key=" + cred.AccessKey + "\n" +
			"api_endpoint=" + apiEndpoint + "\n\n")

		if err != nil {
			panic(err)
		}
	}
}

func (conf *Configure) ParseKeyValue(str string, expr string, errMsg string) string {
	r, err := regexp.Compile(expr)
	if err != nil {
		panic(err)
	}
	match := r.FindStringSubmatch(str)
	if len(match) == 0 {
		log.Fatal(errMsg)
	}
	return match[1]
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

		[profile2]
		key_id=key_id_example_2
		api_key=api_key_example_2
		access_key=access_key_example_2
		api_endpoint=http://localhost:8000

		Args:
			profile: the specific profile in config file
		Returns:
			An instance of Credential
	*/
	usr, err := user.Current()
	homeDir := ""
	if err == nil {
		homeDir = usr.HomeDir
	}
	configPath := path.Join(homeDir + "/.gen3/config")
	cred := Credential{
		KeyId:       "",
		APIKey:      "",
		AccessKey:   "",
		APIEndpoint: "",
	}
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		fmt.Println("No config file found in ~/.gen3/")
		fmt.Println("Run configure command (with a profile if desired) to set up account credentials \n" +
			"Example: ./gen3-client configure --profile=<profile-name> --cred=<path-to-credential/cred.json> --apiendpoint=https://data.mycommons.org")
		return cred
	}
	// If profile not in config file, prompt user to set up config first

	content, err := ioutil.ReadFile(configPath)
	if err != nil {
		panic(err)
	}
	lines := strings.Split(string(content), "\n")

	profileLine := -1
	for i := 0; i < len(lines); i += 6 {
		if lines[i] == "["+profile+"]" {
			profileLine = i
			break
		}
	}

	if profileLine == -1 {
		fmt.Println("Profile not in config file. Need to run \"gen3-client configure --profile=" + profile + " --cred=<path-to-credential/cred.json> --apiendpoint=<api_endpoint_url>\" first")
		return cred
	} else {
		// Read in access key, secret key, endpoint for given profile
		cred.KeyId = conf.ParseKeyValue(lines[profileLine+1], "^key_id=(\\S*)", "key_id not found in profile")
		cred.APIKey = conf.ParseKeyValue(lines[profileLine+2], "^api_key=(\\S*)", "api_key not found in profile")
		cred.AccessKey = conf.ParseKeyValue(lines[profileLine+3], "^access_key=(\\S*)", "access_key not found in profile")
		cred.APIEndpoint = conf.ParseKeyValue(lines[profileLine+4], "^api_endpoint=(\\S*)", "api_endpoint not found in profile")
		return cred
	}
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
