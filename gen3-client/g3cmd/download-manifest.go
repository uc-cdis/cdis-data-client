package g3cmd

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"strings"

	"github.com/spf13/cobra"

	"github.com/uc-cdis/gen3-client/gen3-client/jwt"
)

type ManifestObject struct {
	ObjectID  string `json:"object_id"`
	SubjectID string `json:"subject_id"`
}

func init() {
	var manifest string
	var downloadPath string

	var downloadManifestCmd = &cobra.Command{
		Use:   "download-manifest",
		Short: "download files from a specified manifest",
		Long: `Gets a presigned URL for a file from a GUID and then downloads the specified file.
	Examples: ./gen3-client download-manifest --profile user1 --manifest manifest.tsv --download-path=files/ 
	`,
		Run: func(cmd *cobra.Command, args []string) {

			request := new(jwt.Request)
			configure := new(jwt.Configure)
			function := new(jwt.Functions)

			function.Config = configure
			function.Request = request

			var objects []ManifestObject
			manifestBytes, err := ioutil.ReadFile(manifest)
			if err != nil {
				log.Fatalf("Failed reading manifest %s, %v\n", manifest, err)
			}
			json.Unmarshal(manifestBytes, &objects)

			for _, object := range objects {
				endPointPostfix := "/user/data/download/" + object.ObjectID + "?protocol=s3"
				respURL, err := function.DoRequestWithSignedHeader(profile, "", endPointPostfix)

				if err != nil {
					if strings.Contains(err.Error(), "The provided guid") {
						log.Printf("Download error: %s\n", err)
					} else {
						log.Fatalf("Fatal download error: %s\n", err)
					}
				} else {
					downloadFile(object.ObjectID, downloadPath+"/"+object.ObjectID, respURL)
				}
			}

		},
	}

	downloadManifestCmd.Flags().StringVar(&manifest, "manifest", "", "The manifest file to read from")
	downloadManifestCmd.MarkFlagRequired("manifest")
	downloadManifestCmd.Flags().StringVar(&downloadPath, "download-path", "", "The directory in which to store the downloaded files")
	downloadManifestCmd.MarkFlagRequired("download-path")
	RootCmd.AddCommand(downloadManifestCmd)
}
