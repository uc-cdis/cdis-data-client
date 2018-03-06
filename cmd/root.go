package cmd

import (
	"fmt"
	"os"

	homedir "github.com/mitchellh/go-homedir"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var cfgFile string
var credFile string
var profile string
var uri string
var file_path string
var file_type string
var uuid string

/* RootCmd represents the base command when called without any subcommands */
var RootCmd = &cobra.Command{
	Use:   "cdis-client",
	Short: "CLI with HMAC verification to talk to GDCAPI",
	Long: `Send GET, PUT, POST, DELETE HTTP requests to interact with the GDCAPI
that are signed using HMAC protocol`,
}

/* Execute adds all child commands to the root command sets flags appropriately
   This is called by main.main(). It only needs to happen once to the rootCmd. */
func Execute() {
	if err := RootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	// Define flags and configuration settings.
	RootCmd.PersistentFlags().StringVar(&profile, "profile", "default", "Specify profile to add or edit with --profile user2")
	RootCmd.PersistentFlags().StringVar(&credFile, "cred", "", "Specify cred file --cred credentials.json")
	RootCmd.PersistentFlags().StringVar(&uri, "uri", "", "Specify desired URI with --uri=exampleURI")
	RootCmd.PersistentFlags().StringVar(&file_path, "file", "", "Specify file to upload with --file=~/path/to/file")
	RootCmd.PersistentFlags().StringVar(&file_type, "file_type", "json", "Specify file_type you're uploading with --file_type={json|tsv} (defaults to json)")
	RootCmd.PersistentFlags().StringVar(&uuid, "uuid", "", "Specify the uuid for the data you would like to work with")
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := homedir.Dir()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		// Search config in home directory with name ".cdis-client" (without extension).
		viper.AddConfigPath(home)
		viper.SetConfigName(".cdis-client")
	}

	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		fmt.Println("Using config file:", viper.ConfigFileUsed())
	}
}
