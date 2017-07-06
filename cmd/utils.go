package cmd

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/user"
	"regexp"
	"strings"
)

func parse_config(profile string) (string, string, string) {
	//Look in config file
	usr, _ := user.Current()
	homeDir := usr.HomeDir
	configPath := homeDir + "/.cdis/config"
	if _, err := os.Stat(homeDir + "/.cdis/"); os.IsNotExist(err) {
		fmt.Println("No config file found in ~/.cdis/")
		fmt.Println("Run configure command (with a profile if desired) to set up account credentials")
		return "", "", ""
	}
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		fmt.Println("No config file found in ~/.cdis/")
		fmt.Println("Run configure command (with a profile if desired) to set up account credentials")
		return "", "", ""
	}
	// If profile not in config file, prompt user to set up config first

	content, err := ioutil.ReadFile(configPath)
	if err != nil {
		panic(err)
	}
	lines := strings.Split(string(content), "\n")

	profile_line := -1
	for i := 0; i < len(lines); i += 5 {
		if lines[i] == "["+profile+"]" {
			profile_line = i
			break
		}
	}

	if profile_line == -1 {
		fmt.Println("Profile not in config file. Need to run \"cdis-data-client configure --profile=" + profile + "\" first")
		return "", "", ""
	} else {
		// Read in access key, secret key, endpoint for given profile
		access_key := lines[profile_line+1]
		r, _ := regexp.Compile("access_key=(\\S*)")
		access_key = r.FindStringSubmatch(access_key)[1]

		secret_key := lines[profile_line+2]
		r, _ = regexp.Compile("secret_key=(\\S*)")
		secret_key = r.FindStringSubmatch(secret_key)[1]

		gdcapi_endpoint := lines[profile_line+3]
		r, _ = regexp.Compile("gdcapi_endpoint=(\\S*)")
		gdcapi_endpoint = r.FindStringSubmatch(gdcapi_endpoint)[1]
		return access_key, secret_key, gdcapi_endpoint
	}
}

func read_file(file_path, file_type string) string {
	fmt.Println("file_path")
	fmt.Println(file_path)
	//Look in config file
	var full_file_path string
	if file_path[0] == '~' {
		usr, _ := user.Current()
		homeDir := usr.HomeDir
		full_file_path = homeDir + file_path[1:]
	} else {
		full_file_path = file_path
	}
	fmt.Println("full_file_path")
	fmt.Println(full_file_path)
	if _, err := os.Stat(full_file_path); err != nil {
		fmt.Println("File specified at " + full_file_path + " not found")
		return ""
	}

	content, err := ioutil.ReadFile(full_file_path)
	if err != nil {
		panic(err)
	}
	fmt.Println("content")
	fmt.Println(content)

	content_str := string(content[:])

	if file_type == "json" {
		content_str = strings.Replace(content_str, "\n", "", -1)
	}
	fmt.Println("content_str")
	fmt.Println(content_str)
	return content_str
}
