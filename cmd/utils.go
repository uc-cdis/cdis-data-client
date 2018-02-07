package cmd

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/user"
	"path"
	"regexp"
	"strings"
	"net/url"
	"net/http"
	"bytes"
)

func ParseKeyValue(str string, expr string, errMsg string) (string) {
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

func ParseConfig(profile string) (Credential) {
	//Look in config file
	usr, _ := user.Current()
	homeDir := usr.HomeDir
	configPath := path.Join(homeDir + "/.cdis/config")
	cred := Credential{
		KeyId: "",
		APIKey: "",
		AccessKey: "",
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
		cred.APIKey = ParseKeyValue(lines[profile_line+3], "^access_key=(\\S*)", "access_key not found in profile")
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

type DoRequest func(cred Credential, host *url.URL, contentType string) (*http.Response)

func DoRequestWithSignedHeader(fn DoRequest) (*http.Response){
	cred := ParseConfig(profile)
	if cred.APIKey == "" && cred.AccessKey == "" && cred.APIEndpoint == "" {
		panic("No credential found")
	}
	//TODO: 1. If cred.AccessKey == "",
	//TODO: 2. include cred.APIKey into the request header to refresh cred.AccessKey then write to profile
	//TODO: 3. Else If cred.AccessKey != "",
	//TODO: 4. include cred.AccessKey into the request header and do requesting,
	//TODO: 5. If response says that the AccessKey is expired, repeat step 2 to refresh AccessKey

	contentType := "application/json"
	host, _ := url.Parse(cred.APIEndpoint)
	return fn(cred, host, contentType)
}

func ResponseToString(resp *http.Response) (string) {
	buf := new(bytes.Buffer)
	buf.ReadFrom(resp.Body)
	return buf.String()
}
