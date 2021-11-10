package g3cmd

import (
	"fmt"
	"log"
	"os"

	"github.com/spf13/cobra"
	latest "github.com/tcnksm/go-latest"
	"github.com/uc-cdis/gen3-client/gen3-client/jwt"
	"github.com/uc-cdis/gen3-client/gen3-client/logs"
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
	if gitversion != "" && gitversion != "N/A" {
		githubTag := &latest.GithubTag{
			Owner:      "uc-cdis",
			Repository: "cdis-data-client",
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
