package g3cmd

import (
	"fmt"
	"log"
	"os"

	homedir "github.com/mitchellh/go-homedir"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	latest "github.com/tcnksm/go-latest"
	"github.com/uc-cdis/gen3-client/gen3-client/logs"
)

var cfgFile string
var profile string
var uri string

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
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		homeDir, err := homedir.Dir()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		// Search config in home directory with name ".gen3" (without extension).
		viper.AddConfigPath(homeDir)
		viper.SetConfigName(".gen3")
	}

	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		fmt.Println("Using config file:", viper.ConfigFileUsed())
	}

	logs.Init()
	logs.InitMessageLog(profile)
	logs.SetToBoth()
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
