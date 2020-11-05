package g3cmd

import (
	"log"

	"github.com/spf13/cobra"
)

//Not support yet

var deleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Send DELETE HTTP Request for given URI",
	Long: `Deletes a given URI from the database.
If no profile is specified, "default" profile is used for authentication.`,
	Example: `./gen3-client delete --uri=v0/submission/bpa/test/entities/example_id
	  ./gen3-client delete --profile=user1 --uri=v0/submission/bpa/test/entities/1af1d0ab-efec-4049-98f0-ae0f4bb1bc64`,
	Run: func(cmd *cobra.Command, args []string) {
		log.Fatalf("Not supported!")
		// request := new(jwt.Request)
		// configure := new(jwt.Configure)
		// function := new(jwt.Functions)

		// function.Config = configure
		// function.Request = request

		// fmt.Println(jwt.ResponseToString(
		// 	function.DoRequestWithSignedHeader(RequestDelete, profile, "txt", uri)))
	},
}

func init() {
	//RootCmd.AddCommand(deleteCmd)
}
