package g3cmd

import (
	"log"
	"sort"

	"github.com/spf13/cobra"
	"github.com/uc-cdis/gen3-client/gen3-client/jwt"
	"github.com/uc-cdis/gen3-client/gen3-client/logs"
)

func init() {

	var authCmd = &cobra.Command{
		Use:     "auth",
		Short:   "Return data access privileges from profile",
		Long:    `Gets data access privileges for specified profile.`,
		Example: `./gen3-client auth --profile=<profile-name>`,
		Run: func(cmd *cobra.Command, args []string) {
			// don't initialize transmission logs for non-uploading related commands
			logs.SetToBoth()

			request := new(jwt.Request)
			configure := new(jwt.Configure)
			function := new(jwt.Functions)

			function.Request = request
			function.Config = configure

			host, projectAccess, err := function.CheckPrivileges(profile, "")

			if err != nil {
				log.Fatalf("Fatal authentication error: %s\n", err)
			} else {
				if len(projectAccess) == 0 {
					log.Printf("\nYou don't currently have access to data from any projects at %s\n", host)
				} else {
					log.Printf("\nYou have access to the following project(s) at %s:\n", host)

					// Sort by project name
					projects := make([]string, 0, len(projectAccess))
					for project := range projectAccess {
						projects = append(projects, project)
					}
					sort.Strings(projects)

					for _, project := range projects {
						// Sort by access name
						permissions := projectAccess[project].([]interface{})
						access := make([]string, 0, len(permissions))
						for _, permission := range permissions {
							access = append(access, permission.(string))
						}
						sort.Strings(access)

						log.Printf("%s %s\n", project, access)
					}
				}
			}
			logs.CloseMessageLog()
		},
	}

	authCmd.Flags().StringVar(&profile, "profile", "", "Specify the profile to check your access privileges")
	authCmd.MarkFlagRequired("profile")
	RootCmd.AddCommand(authCmd)
}
