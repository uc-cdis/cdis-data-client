// Copyright © 2017 NAME HERE <EMAIL ADDRESS>
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cmd

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"os"
	"os/user"
	"strings"

	"github.com/spf13/cobra"
)

var profile string

// configureCmd represents the configure command
var configureCmd = &cobra.Command{
	Use:   "configure",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: func(cmd *cobra.Command, args []string) {
		// Prompt user for info
		scanner := bufio.NewScanner(os.Stdin)
		fmt.Print("Access Key: ")
		scanner.Scan()
		accessKey := scanner.Text()
		fmt.Print("Secrete Access Key: ")
		scanner.Scan()
		secretKey := scanner.Text()

		// Store user info in ~/.cdis/config
		usr, _ := user.Current()
		homeDir := usr.HomeDir
		configPath := homeDir + "/.cdis/config"
		if _, err := os.Stat(homeDir + "/.cdis/"); os.IsNotExist(err) {
			os.Mkdir(homeDir+"/.cdis/", os.FileMode(0777))
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
				lines[i+1] = "access_key=" + accessKey
				lines[i+2] = "secret_key=" + secretKey
				found = true
				break
			}
		}

		if found {
			f, err := os.OpenFile(configPath, os.O_WRONLY, 0777)
			if err != nil {
				panic(err)
			}
			defer func() {
				if err := f.Close(); err != nil {
					panic(err)
				}
			}()
			for i := 0; i < len(lines); i++ {
				if lines[i] != "" {
					fmt.Print("newline\n")
					f.WriteString(lines[i] + "\n")
				}
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
			if _, err := f.WriteString("gdcapi_endpoint=IDunnoWhatThisIs" + "\n\n"); err != nil {
				panic(err)
			}
		}

	},
}

func init() {
	RootCmd.AddCommand(configureCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// configureCmd.PersistentFlags().String("foo", "", "A help for foo")
	configureCmd.PersistentFlags().StringVar(&profile, "profile", "default", "example: --profile user2")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// configureCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
