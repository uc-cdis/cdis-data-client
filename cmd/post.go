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
	"net/http"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/uc-cdis/cdis-data-client/gdcHmac"
)

// postCmd represents the post command
var postCmd = &cobra.Command{
	Use:   "post",
	Short: "Send POST HTTP Request to the gdcapi",
	Long: `Sends a POST HTTP Request to make graphql queries stored in 
local json files to the gdcapi. 
If no profile is specified, "default" profile is used for authentication. 

Examples: ./cdis-data-client put --uri=v0/submission/graphql --file=~/Documents/my_grqphql_query.json
	  ./cdis-data-client put --profile=user1 --uri=v0/submission/graphql --file=~/Documents/my_grqphql_query.json
`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("post called")
		access_key, secret_key, gdcapi_endpoint := parse_config(profile)
		if access_key == "" && secret_key == "" && gdcapi_endpoint == "" {
			return
		}
		client := &http.Client{}
		host := strings.TrimPrefix(gdcapi_endpoint, "http://")

		uri = strings.TrimPrefix(uri, "/")

		// Create and send request
		body := bytes.NewBufferString(read_file(file_path, file_type))
		req, _ := http.NewRequest("POST", "http://"+host+"/"+uri, body)
		req.Header.Add("Host", host)
		req.Header.Add("X-Amz-Date", time.Now().UTC().Format("20060102T150405Z"))
		req.Header.Add("Content-Type", "application/"+file_type)

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
	},
}

func init() {
	RootCmd.AddCommand(postCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// postCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// postCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
