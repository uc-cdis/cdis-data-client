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
	"net/http"
	"os"
	"os/user"
	"regexp"
	"strings"

	"github.com/smartystreets/go-aws-auth"
	"github.com/spf13/cobra"
)

var signProfile string

// formatRequest generates ascii representation of a request
func formatRequest(r *http.Request) string {
	// Create return string
	var request []string
	// Add the request string
	url := fmt.Sprintf("%v %v %v", r.Method, r.URL, r.Proto)
	request = append(request, url)
	// Add the host
	request = append(request, fmt.Sprintf("Host: %v", r.Host))
	// Loop through headers
	for name, headers := range r.Header {
		name = strings.ToLower(name)
		for _, h := range headers {
			request = append(request, fmt.Sprintf("%v: %v", name, h))
		}
	}

	// If this is a POST, add post data
	if r.Method == "POST" {
		r.ParseForm()
		request = append(request, "\n")
		request = append(request, r.Form.Encode())
	}
	// Return the request as a string
	return strings.Join(request, "\n")
}

// signCmd represents the sign command
var signCmd = &cobra.Command{
	Use:   "sign",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: func(cmd *cobra.Command, args []string) {
		url := "https://iam.amazonaws.com/?Action=ListRoles&Version=2010-05-08"
		req, _ := http.NewRequest("GET", url, nil)

		usr, _ := user.Current()
		dir := usr.HomeDir
		file, err := os.Open(dir + "/.cdis/config")
		if err != nil {
			panic(err)
		}
		configScanner := bufio.NewScanner(file)
		var accessKey string
		var secretKey string
		r, _ := regexp.Compile("=.*")
		for configScanner.Scan() {
			if configScanner.Text() == "["+profile+"]" {
				configScanner.Scan()
				accessKey = r.FindString(configScanner.Text())
				accessKey = strings.TrimPrefix(accessKey, "=")
				configScanner.Scan()
				secretKey = r.FindString(configScanner.Text())
				secretKey = strings.TrimPrefix(secretKey, "=")
			}
		}
		fmt.Println("accessKey: " + accessKey)
		fmt.Println("secretKey: " + secretKey)

		awsauth.Sign4(req, awsauth.Credentials{
			AccessKeyID:     accessKey,
			SecretAccessKey: secretKey,
		})
		fmt.Println(formatRequest(req))
	},
}

func init() {
	RootCmd.AddCommand(signCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// signCmd.PersistentFlags().String("foo", "", "A help for foo")
	configureCmd.PersistentFlags().StringVar(&signProfile, "signProfile", "default", "example: --signProfile user2")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// signCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
