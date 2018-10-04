package g3cmd

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/spf13/cobra"

	"github.com/uc-cdis/gen3-client/gen3-client/jwt"
)

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

			endPointPostfix := "/user/data/download/" + manifest

			respUrl, err := function.DoRequestWithSignedHeader(profile, "", endPointPostfix)

			if err != nil {
				log.Fatalf("Download error: %s\n", err)
			} else {
				respDown, err := http.Get(respUrl)
				if err != nil {
					log.Fatalf("Download error: %s\n", err)
				}
				out, err := os.Create(downloadPath)
				if err != nil {
					log.Fatalf(err.Error())
				}
				defer out.Close()
				defer respDown.Body.Close()
				_, err = io.Copy(out, respDown.Body)
				if err != nil {
					panic(err)
				}

				fmt.Printf("Successfully downloaded %s to %s!\n", manifest, downloadPath)
			}

		},
	}

	downloadManifestCmd.Flags().StringVar(&manifest, "manifest", "", "The manifest file to read from")
	downloadManifestCmd.MarkFlagRequired("manifest")
	downloadManifestCmd.Flags().StringVar(&downloadPath, "download-path", "", "The directory in which to store the downloaded files")
	downloadManifestCmd.MarkFlagRequired("download-path")
	RootCmd.AddCommand(downloadManifestCmd)
}
