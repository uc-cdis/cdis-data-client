package cmd

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"os"
	"os/user"
	"path"
	"strings"

	"github.com/spf13/cobra"
)

var configureCmd = &cobra.Command{
	Use:   "configure",
	Short: "Add or modify a configuration profile to your config file",
	Long: `Configuration file located at ~/.cdis/config
Prompts for access_key, secret_key, and gdcapi endpoint
If a field is left empty, the existing value (if it exists) will remain unchanged
If no profile is specified, "default" profile is used

Examples: ./cdis-data-client config
	  ./cdis-data-client config --profile=user1`,
	Run: func(cmd *cobra.Command, args []string) {
		// Prompt user for info
		scanner := bufio.NewScanner(os.Stdin)
		fmt.Print("Access Key: ")
		scanner.Scan()
		accessKey := scanner.Text()
		fmt.Print("Secret Key: ")
		scanner.Scan()
		secretKey := scanner.Text()
		fmt.Print("API endpoint: ")
		scanner.Scan()
		api_endpoint := scanner.Text()

		// Store user info in ~/.cdis/config
		usr, err := user.Current()
		if err != nil {
			panic(err)
		}
		homeDir := usr.HomeDir
		configPath := path.Join(homeDir + "/.cdis/config")
		if _, err := os.Stat(path.Join(homeDir + "/.cdis/")); os.IsNotExist(err) {
			os.Mkdir(path.Join(homeDir+"/.cdis/"), os.FileMode(0777))
			os.Create(configPath)
		}
		if _, err := os.Stat(configPath); os.IsNotExist(err) {
			os.Create(configPath)
		}

		content, err := ioutil.ReadFile(configPath)
		if err != nil {
			panic(err)
		}
		lines := strings.Split(string(content), "\n")

		found := false
		for i := 0; i < len(lines); i += 5 {
			if lines[i] == "["+profile+"]" {
				if accessKey != "" {
					lines[i+1] = "access_key=" + accessKey
				}
				if secretKey != "" {
					lines[i+2] = "secret_key=" + secretKey
				}
				if api_endpoint != "" {
					lines[i+3] = "api_endpoint=" + api_endpoint
				}
				found = true
				break
			}
		}

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

			if _, err := f.WriteString("[" + profile + "]\n"); err != nil {
				panic(err)
			}

			if _, err := f.WriteString("access_key=" + accessKey + "\n"); err != nil {
				panic(err)
			}
			if _, err := f.WriteString("secret_key=" + secretKey + "\n"); err != nil {
				panic(err)
			}
			if _, err := f.WriteString("api_endpoint=" + api_endpoint + "\n\n"); err != nil {
				panic(err)
			}
		}

	},
}

func init() {
	RootCmd.AddCommand(configureCmd)
}
