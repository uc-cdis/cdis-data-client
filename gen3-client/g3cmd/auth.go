package g3cmd

import (
	"encoding/json"
	"log"
	"sort"

	"github.com/spf13/cobra"
	"github.com/uc-cdis/gen3-client/gen3-client/logs"
)

func init() {

	var authCmd = &cobra.Command{
		Use:     "auth",
		Short:   "Return resource access privileges from profile",
		Long:    `Gets resource access privileges for specified profile.`,
		Example: `./gen3-client auth --profile=<profile-name>`,
		Run: func(cmd *cobra.Command, args []string) {
			// don't initialize transmission logs for non-uploading related commands
			logs.SetToBoth()
			gen3Interface := NewGen3Interface()
				profileConfig, err := conf.ParseConfig(profile)
			if err != nil {
				log.Fatalf("Fatal config parse error: %s\n", err)
			}

			host, resourceAccess, err := gen3Interface.CheckPrivileges(&profileConfig)

			if err != nil {
				log.Fatalf("Fatal authentication error: %s\n", err)
			} else {
				if len(resourceAccess) == 0 {
					log.Printf("\nYou don't currently have access to any resources at %s\n", host)
				} else {
					log.Printf("\nYou have access to the following resource(s) at %s:\n", host)

					// Sort by resource name
					resources := make([]string, 0, len(resourceAccess))
					for resource := range resourceAccess {
						resources = append(resources, resource)
					}
					sort.Strings(resources)

					for _, project := range resources {
						// Sort by access name if permissions are from Fence
						permissions := resourceAccess[project].([]interface{})
						_, isFencePermission := permissions[0].(string)
						if isFencePermission {
							access := make([]string, 0, len(permissions))
							for _, permission := range permissions {
								access = append(access, permission.(string))
							}
							sort.Strings(access)
							log.Printf("%s %s\n", project, access)
						} else {
							// Permissions from Arborist already sorted, just pretty print them
							marshalledPermissions, err := json.MarshalIndent(permissions, "", "  ")
							if err != nil {
								log.Printf("%s (error occurred when marshalling permissions): %s\n", project, err)
							}
							log.Printf("%s %s\n", project, marshalledPermissions)
						}
					}
				}
			}
			err = logs.CloseMessageLog()
			if err != nil {
				log.Println(err.Error())
			}
		},
	}

	authCmd.Flags().StringVar(&profile, "profile", "", "Specify the profile to check your access privileges")
	authCmd.MarkFlagRequired("profile") // nolint: errcheck
	RootCmd.AddCommand(authCmd)
}
