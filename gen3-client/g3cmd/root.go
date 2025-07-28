package g3cmd

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	latest "github.com/tcnksm/go-latest"
	"github.com/calypr/gen3-client/gen3-client/jwt"
	"github.com/calypr/gen3-client/gen3-client/logs"
)

var profile string
var profileConfig jwt.Credential

// RootCmd represents the base command when called without any subcommands
var RootCmd = &cobra.Command{
	Use:     "gen3-client",
	Short:   "Use the gen3-client to interact with a Gen3 Data Commons",
	Long:    "Gen3 Client for downloading, uploading and submitting data to data commons.\ngen3-client version: " + gitversion + ", commit: " + gitcommit,
	Version: gitversion,
}

// Execute adds all child commands to the root command sets flags appropriately
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := RootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	// Define flags and configuration settings.
	RootCmd.PersistentFlags().StringVar(&profile, "profile", "", "Specify profile to use")
	_ = RootCmd.MarkFlagRequired("profile")
}

func initConfig() {

	logs.Init()
	logs.InitMessageLog(profile)
	logs.SetToBoth()

	// init local config file
	err := conf.InitConfigFile()
	if err != nil {
		log.Fatalln("Error occurred when trying to init config file: " + err.Error())
	}

	// version checker
	if os.Getenv("GEN3_CLIENT_VERSION_CHECK") != "false" &&
	gitversion != "" && gitversion != "N/A" {
		githubTag := &latest.GithubTag{
			Owner:      "uc-cdis",
			Repository: "cdis-data-client",
			TagFilterFunc: func(versionTag string) bool {
				// only assume a version tag to be valid version tag if it has either 2 or 3 "." in it
				// so tags like "whatever" or "new.123.release" won't interfere
				gitversionSlice := strings.Split(gitversion, ".")
				versionTagSlice := strings.Split(versionTag, ".")
				// if gitversion is sematic version number, ignore tags that don't have 3 "."
				// if gitversion is monthly release version number, ignore tags that don't have 2 "."
				if (len(gitversionSlice) == 3 && len(versionTagSlice) != 3) || (len(gitversionSlice) == 2 && len(versionTagSlice) != 2) {
					return false
				}
				for _, s := range versionTagSlice {
					_, err := strconv.Atoi(s)
					if err != nil {
						return false
					}
				}
				return true
			},
		}
		res, err := latest.Check(githubTag, gitversion)
		if err != nil {
			log.Println("Error occurred when checking for latest version: " + err.Error())
		} else if res.Outdated {
			log.Println("A new version of gen3-client is available! The latest version is " + res.Current + ". You are using version " + gitversion)
			log.Println("Please download the latest gen3-client release from https://github.com/uc-cdis/cdis-data-client/releases/latest")
		}
	}
	logs.SetToMessageLog()
}
