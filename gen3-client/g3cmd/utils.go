package g3cmd

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/user"
	"path"
	"regexp"
	"strings"
)

func parse_config(profile string) (string, string, string) {
	//Look in config file
	usr, _ := user.Current()
	homeDir := usr.HomeDir
	configPath := path.Join(homeDir + "/.gen3/config")
	if _, err := os.Stat(path.Join(homeDir + "/.gen3/")); os.IsNotExist(err) {
		fmt.Println("No config file found in ~/.gen3/")
		fmt.Println("Run configure command (with a profile if desired) to set up account credentials")
		return "", "", ""
	}
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		fmt.Println("No config file found in ~/.gen3/")
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
		fmt.Println("Profile not in config file. Need to run \"gen3-client configure --profile=" + profile + " --cred path_to_credential.json\" first")
		return "", "", ""
	} else {
		// Read in access key, secret key, endpoint for given profile
		access_key := lines[profile_line+1]
		r, err := regexp.Compile("^access_key=(\\S*)")
		if err != nil {
			panic(err)
		}
		match := r.FindStringSubmatch(access_key)
		if len(match) == 0 {
			log.Fatal("access_key not found in profile")
		}
		access_key = match[1]

		secret_key := lines[profile_line+2]
		r, err = regexp.Compile("^secret_key=(\\S*)")
		if err != nil {
			panic(err)
		}
		match = r.FindStringSubmatch(secret_key)
		if len(match) == 0 {
			log.Fatal("secret_key not found in profile")
		}
		secret_key = match[1]

		api_endpoint := lines[profile_line+3]
		r, err = regexp.Compile("^api_endpoint=(\\S*)")
		if err != nil {
			panic(err)
		}
		match = r.FindStringSubmatch(api_endpoint)
		if len(match) == 0 {
			log.Fatal("api_endpoint not found in profile")
		}
		api_endpoint = match[1]
		return access_key, secret_key, api_endpoint
	}
}
