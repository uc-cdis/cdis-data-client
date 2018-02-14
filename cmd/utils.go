package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/user"
	"path"
	"regexp"
	"strings"
)

type APIKeyStruct struct {
	Api_key string
	Key_id  string
}

type AccessTokenStruct struct {
	Access_token string
}

func ParseKeyValue(str string, expr string, errMsg string) string {
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

func ParseConfig(profile string) Credential {
	//Look in config file
	usr, _ := user.Current()
	homeDir := usr.HomeDir
	configPath := path.Join(homeDir + "/.cdis/config")
	cred := Credential{
		KeyId:       "",
		APIKey:      "",
		AccessKey:   "",
		APIEndpoint: "",
	}
	if _, err := os.Stat(path.Join(homeDir + "/.cdis/")); os.IsNotExist(err) {
		fmt.Println("No config file found in ~/.cdis/")
		fmt.Println("Run configure command (with a profile if desired) to set up account credentials")
		return cred
	}
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		fmt.Println("No config file found in ~/.cdis/")
		fmt.Println("Run configure command (with a profile if desired) to set up account credentials")
		return cred
	}
	// If profile not in config file, prompt user to set up config first

	content, err := ioutil.ReadFile(configPath)
	if err != nil {
		panic(err)
	}
	lines := strings.Split(string(content), "\n")

	profile_line := -1
	for i := 0; i < len(lines); i += 6 {
		if lines[i] == "["+profile+"]" {
			profile_line = i
			break
		}
	}

	if profile_line == -1 {
		fmt.Println("Profile not in config file. Need to run \"cdis-data-client configure --profile=" + profile + "\" first")
		return cred
	} else {
		// Read in access key, secret key, endpoint for given profile
		cred.KeyId = ParseKeyValue(lines[profile_line+1], "^key_id=(\\S*)", "key_id not found in profile")
		cred.APIKey = ParseKeyValue(lines[profile_line+2], "^api_key=(\\S*)", "api_key not found in profile")
		cred.AccessKey = ParseKeyValue(lines[profile_line+3], "^access_key=(\\S*)", "access_key not found in profile")
		cred.APIEndpoint = ParseKeyValue(lines[profile_line+4], "^api_endpoint=(\\S*)", "api_endpoint not found in profile")
		return cred
	}
}

func ReadFile(file_path, file_type string) string {
	//Look in config file
	var full_file_path string
	if file_path[0] == '~' {
		usr, _ := user.Current()
		homeDir := usr.HomeDir
		full_file_path = homeDir + file_path[1:]
	} else {
		full_file_path = file_path
	}
	if _, err := os.Stat(full_file_path); err != nil {
		fmt.Println("File specified at " + full_file_path + " not found")
		return ""
	}

	content, err := ioutil.ReadFile(full_file_path)
	if err != nil {
		panic(err)
	}

	content_str := string(content[:])

	if file_type == "json" {
		content_str = strings.Replace(content_str, "\n", "", -1)
	}
	return content_str
}

func TryReadFile(filePath string) ([]byte, error) {
	if _, err := os.Stat(path.Dir(filePath)); os.IsNotExist(err) {
		os.Mkdir(path.Join(path.Dir(filePath)), os.FileMode(0777))
		os.Create(filePath)
	}
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		os.Create(filePath)
	}

	return ioutil.ReadFile(filePath)
}

func MakeARequest(client *http.Client, method string, path string, headers map[string]string, body *bytes.Buffer) (*http.Response, error) {
	req, err := http.NewRequest(method, path, body)
	if err != nil {
		return nil, err
	}
	for k, v := range headers {
		req.Header.Add(k, v)
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	return resp, nil

}

func GetAPIKey(client *http.Client, path string) (APIKeyStruct, error) {
	body := bytes.NewBufferString("'scope': ['data', 'user']")
	headers := make(map[string]string)
	headers["Content-Type"] = "application/x-www-form-urlencode"
	resp, err := MakeARequest(client, "POST", path, headers, body)
	var m APIKeyStruct
	if err != nil {
		return m, err
	}
	err = json.Unmarshal(ResponseToBytes(resp), &m)
	if err != nil {
		return m, err
	}
	return m, nil

}

func GetAccessKey(client *http.Client, path string, apiKey string) (AccessTokenStruct, error) {
	body := bytes.NewBufferString("{\"api_key\": \"" + apiKey + "\"}")
	headers := make(map[string]string)
	headers["Content-Type"] = "application/json"
	resp, err := MakeARequest(client, "POST", path, headers, body)
	var m AccessTokenStruct
	if err != nil {
		return m, err
	}
	err = json.Unmarshal(ResponseToBytes(resp), &m)
	if err != nil {
		return m, err
	}
	return m, nil
}

func Requesting(cred Credential, host *url.URL, contentType string) *http.Response {
	return nil
}

type DoRequest func(cred Credential, host *url.URL, contentType string) *http.Response

func DoRequestWithSignedHeader(fn DoRequest) *http.Response {

	cred := ParseConfig(profile)
	if cred.APIKey == "" && cred.AccessKey == "" && cred.APIEndpoint == "" {
		panic("No credential found")
	}

	client := &http.Client{}

	if cred.AccessKey == "" {

		//Include cred.APIKey into the request header to refresh cred.AccessKey then write to profile
		accessKeyStruct, err := GetAccessKey(client, cred.APIEndpoint+"/credentials/cdis/access_token", cred.APIKey)
		if err != nil {
			return nil
		}
		cred.AccessKey = accessKeyStruct.Access_token

		usr, _ := user.Current()
		homeDir := usr.HomeDir
		configPath := path.Join(homeDir + "/.cdis/config")
		content := ReadFile(configPath, "")
		print("keyid", cred.KeyId)
		UpdateConfigFile(cred, []byte(content), cred.APIEndpoint, configPath)
	}

	contentType := "application/json"
	host, _ := url.Parse(cred.APIEndpoint)
	return fn(cred, host, contentType)
}

func ResponseToString(resp *http.Response) string {
	buf := new(bytes.Buffer)
	buf.ReadFrom(resp.Body)
	return buf.String()
}

func ResponseToBytes(resp *http.Response) []byte {
	strBuf := ResponseToString(resp)
	strBuf = strings.Replace(strBuf, "\n", "", -1)
	return []byte(strBuf)
}
