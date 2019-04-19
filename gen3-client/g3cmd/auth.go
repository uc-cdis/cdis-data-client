package g3cmd

import (
	"fmt"
	"log"
	"sort"

	"github.com/spf13/cobra"
	"github.com/uc-cdis/gen3-client/gen3-client/jwt"
)

func init() {

	var authCmd = &cobra.Command{
		Use:     "auth",
		Short:   "Return data access priveleges from profile",
		Long:    `Gets data access priveleges for specified profile.`,
		Example: `./gen3-client auth --profile=<profile-name>`,
		Run: func(cmd *cobra.Command, args []string) {

			request := new(jwt.Request)
			configure := new(jwt.Configure)
			function := new(jwt.Functions)

			function.Request = request
			function.Config = configure

			endPointPostfix := "/user/user" // Information about current user

			host, project_access, err := function.CheckPrivileges(profile, "", endPointPostfix, "application/json", nil)

			if err != nil {
				log.Fatalf("Fatal authentication error: %s\n", err)
			} else {
				if len(project_access) == 0 {
					fmt.Printf("\nYou don't currently have access to data from any projects at %s\n", host)
				} else {
					fmt.Printf("\nYou have access to the following project(s) at %s:\n", host)

					// Sort by project name
					projects := make([]string, 0, len(project_access))
					for project := range project_access {
						projects = append(projects, project)
					}
					sort.Strings(projects)

					for _, project := range projects {
						fmt.Printf("%s %s\n", project, project_access[project])
					}
				}
			}
		},
	}

	authCmd.Flags().StringVar(&profile, "profile", "", "Specify the profile to check your access privileges")
	authCmd.MarkFlagRequired("profile")
	RootCmd.AddCommand(authCmd)
}
