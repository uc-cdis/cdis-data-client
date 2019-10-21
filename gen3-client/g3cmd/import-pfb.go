package g3cmd

import (
	"encoding/json"
	"github.com/spf13/cobra"
	"github.com/uc-cdis/gen3-client/gen3-client/jwt"
)

const SowerJobEndpoint = "/job"
const SowerJobDispatchEndpoint = SowerJobEndpoint + "/dispatch"

type SowerJobInputObject struct {
	Url string `json:"url"`
}

type SowerJobDispatchImportObject struct {
	Action string `json:"action"`
	Input SowerJobInputObject `json:"input"`
}

func init() {
	var importPFBFileURL string
	var uploadCmd = &cobra.Command{
		Use:   "import-pfb",
		Short: "Import PFB to the Data Commons.",
		Long:  `Import PFB from pre-signed URLs for PFB file into the Data Commons.`,
		Example: "For import PFB file to Data Commons:\n" +
			"./gen3-client import-fb --profile=<profile-name> --pfb-file=<presigned-url>",
		Run: func(cmd *cobra.Command, args []string) {
			request := new(jwt.Request)
			configure := new(jwt.Configure)
			function := new(jwt.Functions)

			function.Config = configure
			function.Request = request

			action := SowerJobDispatchImportObject{Action: "import", Input: SowerJobInputObject{Url: importPFBFileURL}}
			objectBytes, _ := json.Marshal(action)

			function.DoRequestWithSignedHeader(profile, "", SowerJobDispatchEndpoint, "application/json", objectBytes)
		},
	}

	uploadCmd.Flags().StringVar(&profile, "profile", "", "Specify profile to use")
	uploadCmd.MarkFlagRequired("profile")
	uploadCmd.Flags().StringVar(&importPFBFileURL, "pfb-file", "", "Pre-signed URL for PFB file to import.")
	uploadCmd.MarkFlagRequired("pfb-file")
	RootCmd.AddCommand(uploadCmd)
}
