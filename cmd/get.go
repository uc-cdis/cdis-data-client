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
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/user"
	"regexp"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/uc-cdis/cdis-data-client/gdcHmac"
)

var uri string

// getCmd represents the get command
var getCmd = &cobra.Command{
	Use:   "get",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: func(cmd *cobra.Command, args []string) {
		//Look in config file
		usr, _ := user.Current()
		homeDir := usr.HomeDir
		configPath := homeDir + "/.cdis/config"
		if _, err := os.Stat(homeDir + "/.cdis/"); os.IsNotExist(err) {
			fmt.Println("No config file found in ~/.cdis/")
			fmt.Println("Run configure command (with a profile if desired) to set up account credentials")
			return
		}
		if _, err := os.Stat(configPath); os.IsNotExist(err) {
			fmt.Println("No config file found in ~/.cdis/")
			fmt.Println("Run configure command (with a profile if desired) to set up account credentials")
			return
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
			return
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

			client := &http.Client{}
			host := strings.TrimPrefix(gdcapi_endpoint, "http://")

			uri = strings.TrimPrefix(uri, "/")

			// Create and send request
			req, _ := http.NewRequest("GET", "http://"+host+"/"+uri, nil)
			req.Header.Add("Host", host)
			req.Header.Add("X-Amz-Date", time.Now().UTC().Format("20060102T150405Z"))

			signed_req := gdcHmac.Sign(req, gdcHmac.Credentials{AccessKeyID: access_key, SecretAccessKey: secret_key}, "submission")

			// Display what came back
			resp, err := client.Do(signed_req)
			if err != nil {
				panic(err)
			}
			buf := new(bytes.Buffer)
			buf.ReadFrom(resp.Body)
			s := buf.String()
			fmt.Println(s)
		}
	},
}

func init() {
	RootCmd.AddCommand(getCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// getCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// getCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
	getCmd.Flags().StringVar(&uri, "uri", "", "Specify desired URI with --uri=exampleURI")
}
