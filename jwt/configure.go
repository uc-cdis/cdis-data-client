package jwt

//go:generate mockgen -destination=mocks/mock_configure.go -package=mocks github.com/uc-cdis/cdis-data-client/jwt ConfigureInterface

import (
	"bufio"
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
	ParseUrl() string
	ReadLines(Credential, []byte, string, string) ([]string, bool)
	UpdateConfigFile(Credential, []byte, string, string, string)
	ParseKeyValue(str string, expr string, errMsg string) string
	ParseConfig(profile string) Credential
}

func (conf *Configure) ReadFile(file_path string, file_type string) string {
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

func (conf *Configure) ParseUrl() string {
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
	usr, err := user.Current()
	if err != nil {
		panic(err)
	}
	homeDir := usr.HomeDir
	configPath := path.Join(homeDir + "/.cdis/config")
	content, err := conf.TryReadFile(configPath)
	return configPath, content, err
}

func (conf *Configure) ReadLines(cred Credential, configContent []byte, apiEndpoint string, profile string) ([]string, bool) {
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
		Look profile in config file. The config file is a text file located at ~/.cdis directory. It can
		contain more than 1 profile. If there is no profile found, the user is asked to run a command to
		create the profile

		The format of config file is described as following

		[profile1]
		key_id=b1c1f8c4-aa68-4dae-b58f-be19871f7d0c
		api_key=eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCIsImtpZCI6ImtleS0wMSJ9.eyJhdWQiOlsiZmVuY2UiLCJvcGVuaWQiXSwiaXNzIjoiaHR0cDovL2xvY2FsaG9zdCIsImp0aSI6ImIxYzFmOGM0LWFhNjgtNGRhZS1iNThmLWJlMTk4
		NzFmN2QwYyIsImV4cCI6MTUyMTE3MDk5OCwiaWF0IjoxNTE4NTgyNTk4LCJwdXIiOiJhcGlfa2V5Iiwic3ViIjoiMTIxMCJ9.jhdi8j8Ngbouz
		1qahlTgs_BQL5gvQyhLh3ilG2rjQYOFDEayOjXXqKD60VvD1Ln_jeiWrh9nATHoLPWExyFDaTslp9yD6nbh5UJthDWhx3yc3XV6csd_BSRcLqHea2220r_thHMtYd8mc-zp
		k2EMsGVcJOZ7z3QrPk8zP9pLcGWx7FOZM-m6CqMj-quXJRJm7Q1X4SvpWzi8ZtMUHCO-a_y8BeENlj90nVSLqo5068n34OsDVsjAsdPFqexBz6BuMWHXzrO2xLi7BZfqFC87fqF
		gY9Al1M03F1f-BhAjvlTNAgR4lXkJCpJ5vNh_rGx2j9z54_WzykXwGjCHc-d5bA
		access_key=V4cCI6MTUyMTE3MDk5OCwiaWF0IjoxNTE4NTgyNTk4LCJwdXIiOiJhcGlfa2V5Iiwic3ViIjoiMTIxMCJ9.jhdi8j8Ngbouz
		1qahlTgs_BQL5gvQyhLh3ilG2rjQYOFDEayOjXXqKD60VvD1Ln_jeiWrh9nATHoLPWExyFDaTslp9yD6nbh5UJthDWhx3yc3XV6csd_BSRcLqHea2220r_thHMtYd8mc-zp
		k2EMsGVcJOZ7z3QrPk8zP9pLcGWx7FOZM-m6CqMj-quXJRJm7Q1X4SvpWzi8ZtMUHCO-a_1NiIsInR5cCI6IkpXVCIsImtpZCI6ImtleS0wMSJ9.eyJhdWQiOlsiZmVuY2UiLCJvcGVuaWQiXSwiaXNzIjoi
		aHR0cDovL2xvY2FsaG9zdCIsImp0aSI6ImIxYzFmOGM0LWFhNjgtNGRhZS1iNThmLWJlMTk4NzFmN2QwYyIsImV4cCI6MTUyMTE3MDk5OCwiaWF0IjoxNTE4NTgyNTk4LC
		api_endpoint=http://localhost:8000

		[profile2]
		key_id=b267f8c4-44gr-4dae-b58f-be19871f7d0c
		api_key=dfdf.eyJhdWQiOlsiZmVuY2UiLCJvcGVuaWQiXSwiaXNzIjoiaHR0cDovL2xvY2FsaG9zdCIsImp0aSI6ImIxYzFdfdfmOGM0LWFhNjgtNGRhZS1iNThmLWJlMTk4
		NzFmN2QwYyIsImV4cCI6MTUyMTE3MDk5OCwiaWF0IjoxNTE4NTgyNTk4LCJwdXIiOiJhcGlfa2V5Iiwic3ViIjoiMTIxMCJ9.jhdi8j8Ngbouz
		1qahlTgs_BQL5gvQyhLh3ilG2rjQYOFDEayOjXXqKD60VvD1Ln_jeiWrh9nATHoLPWExyFDaTslp9yD6nbh5UJthDWhx3yc3XV6csd_BSRcLqHea2220r_thHMtYd8mc-zp
		k2EMsGVcJOZ7z3QrPk8zP9pLcGdfdfWx7FOZM-m6CqMj-quXJRJm7Q1X4SvpWzi8ZtMUHCO-a_y8BeENlj90nVSLqo5068n34OsDVsjAsdPFqexBz6BuMWHXzrO2xLi7BZfqFC87fqF
		gY9Al1M03F1f-BhAjvlTNAgR4lXkJCpJ5vNh_rGx2j9z54_WzykXwGjCHc-d5bA
		access_key=V4cCI6MTUyMTE3MDk5OCwiaWF0IjoxNTE4NTgyNTk4LCJwdXIiOiJhcGlfa2V5Iiwic3ViIjoiMTIxMCJ9.jhdi8j8Ngbouz
		1qahlTgs_BQL5gvQyhLh3ilG2rjQYOFDEayOjXXqKD60VvD1Ln_jeiWrh9nATHoLPWExyFDaTslp9yD6nbh5UJthDWhx3yc3XV6csd_BSRcLqHea2220r_thHMtYd8mc-zp
		k2EMsGVcJOZ7z3QrPk8zP9pLcGWx7FOZM-m6CqMj-quXJRJm7Q1X4SvpWzi8ZtMUHCO-a_1NiIsInR5cCI6IkpXVCIsImtpZCI6ImtleS0wMSJ9.eyJhdWQiOlsiZmVuY2UiLCJvcGVuaWQiXSwiaXNzIjoi
		aHR0cDovL2xvY2FsaG9zdCIsImp0aSI6ImIxYzFmOGM0LWFhNjgtNGRhZS1iNThmLWJlMTk4
		NzFmN2QwYyIsImV4cCI6MTUyMTE3MDk5OCwiaWF0IjoxNTE4NTgyNTk4LC
		api_endpoint=http://localhost:8000

		Args:
			profile: the specific profile in config file
		Returns:
			An instance of Credential


	*/
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
		cred.KeyId = conf.ParseKeyValue(lines[profile_line+1], "^key_id=(\\S*)", "key_id not found in profile")
		cred.APIKey = conf.ParseKeyValue(lines[profile_line+2], "^api_key=(\\S*)", "api_key not found in profile")
		cred.AccessKey = conf.ParseKeyValue(lines[profile_line+3], "^access_key=(\\S*)", "access_key not found in profile")
		cred.APIEndpoint = conf.ParseKeyValue(lines[profile_line+4], "^api_endpoint=(\\S*)", "api_endpoint not found in profile")
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
