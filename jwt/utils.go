package jwt

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
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

type UtilInterface interface {
	ParseKeyValue(string, string, string) string
	ParseConfig(string) Credential
	TryReadFile(string) ([]byte, error)
	ResponseToString(*http.Response) string
	ResponseToBytes(*http.Response) []byte
}

type Utils struct{}

func (utils *Utils) ParseKeyValue(str string, expr string, errMsg string) string {
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

func (utils *Utils) ParseConfig(profile string) Credential {
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
		cred.KeyId = utils.ParseKeyValue(lines[profile_line+1], "^key_id=(\\S*)", "key_id not found in profile")
		cred.APIKey = utils.ParseKeyValue(lines[profile_line+2], "^api_key=(\\S*)", "api_key not found in profile")
		cred.AccessKey = utils.ParseKeyValue(lines[profile_line+3], "^access_key=(\\S*)", "access_key not found in profile")
		cred.APIEndpoint = utils.ParseKeyValue(lines[profile_line+4], "^api_endpoint=(\\S*)", "api_endpoint not found in profile")
		return cred
	}
}

func (utils *Utils) TryReadFile(filePath string) ([]byte, error) {
	if _, err := os.Stat(path.Dir(filePath)); os.IsNotExist(err) {
		os.Mkdir(path.Join(path.Dir(filePath)), os.FileMode(0777))
		os.Create(filePath)
	}
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		os.Create(filePath)
	}

	return ioutil.ReadFile(filePath)
}

func (utils *Utils) ResponseToString(resp *http.Response) string {
	buf := new(bytes.Buffer)
	buf.ReadFrom(resp.Body)
	return buf.String()
}

func (utils *Utils) ResponseToBytes(resp *http.Response) []byte {
	strBuf := utils.ResponseToString(resp)
	strBuf = strings.Replace(strBuf, "\n", "", -1)
	return []byte(strBuf)
}
