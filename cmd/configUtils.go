package cmd

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"os/user"
	"path"
	"strings"
)

type Credential struct {
	KeyId       string
	APIKey      string
	AccessKey   string
	APIEndpoint string
}

func ReadCredentials(filePath string) Credential {
	var configuration Credential
	jsonContent := ReadFile(filePath, "json")
	jsonContent = strings.Replace(jsonContent, "key_id", "KeyId", -1)
	jsonContent = strings.Replace(jsonContent, "api_key", "APIKey", -1)
	err := json.Unmarshal([]byte(jsonContent), &configuration)
	if err != nil {
		fmt.Println("Cannot read json file: " + err.Error())
		os.Exit(1)
	}
	return configuration
}

func ParseUrl() string {
	scanner := bufio.NewScanner(os.Stdin)
	fmt.Print("API endpoint: ")
	scanner.Scan()
	apiEndpoint := scanner.Text()
	parsed_url, err := url.Parse(apiEndpoint)
	if err != nil {
		panic(err)
	}
	if parsed_url.Host == "" {
		fmt.Print("Invalid endpoint. A valid endpoint looks like: https://www.tests.com\n")
		os.Exit(1)
	}
	return apiEndpoint
}

func TryReadConfigFile() (string, []byte, error) {
	usr, err := user.Current()
	if err != nil {
		panic(err)
	}
	homeDir := usr.HomeDir
	configPath := path.Join(homeDir + "/.cdis/config")
	content, err := TryReadFile(configPath)
	return configPath, content, err
}

func ReadLines(cred Credential, configContent []byte, apiEndpoint string) ([]string, bool) {
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

func UpdateConfigFile(cred Credential, configContent []byte, apiEndpoint string, configPath string) {
	lines, found := ReadLines(cred, configContent, apiEndpoint)
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
