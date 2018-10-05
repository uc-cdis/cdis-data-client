package g3cmd

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"time"

	"github.com/cavaliercoder/grab"
	"github.com/spf13/cobra"

	"github.com/uc-cdis/gen3-client/gen3-client/jwt"
)

type ManifestObject struct {
	ObjectID  string `json:"object_id"`
	SubjectID string `json:"subject_id"`
}

func batchDownload(numParallel int, reqs []*grab.Request) {

	client := grab.NewClient()
	respch := client.DoBatch(numParallel, reqs...)

	t := time.NewTicker(200 * time.Millisecond)

	completed := 0
	inProgress := 0
	responses := make([]*grab.Response, 0)
	for completed < len(reqs) {
		select {
		case resp := <-respch:
			if resp != nil {
				responses = append(responses, resp)
			}

		case <-t.C:
			if inProgress > 0 {
				fmt.Printf("\033[%dA\033[K", inProgress)
			}

			for i, resp := range responses {
				if resp != nil && resp.IsComplete() {
					if resp.Err() != nil {
						fmt.Fprintf(os.Stderr, "Error downloading %s: %v\n", resp.Request.URL(), resp.Err())
					} else {
						fmt.Printf("Finished %s %d / %d bytes (%d%%)\n", resp.Filename, resp.BytesComplete(), resp.Size, int(100*resp.Progress()))
					}

					responses[i] = nil
					completed++
				}
			}

			inProgress = 0
			for _, resp := range responses {
				if resp != nil {
					inProgress++
					fmt.Printf("Downloading %s %d / %d bytes (%d%%)\033[K\n", resp.Filename, resp.BytesComplete(), resp.Size, int(100*resp.Progress()))
				}
			}
		}
	}

	t.Stop()

	fmt.Printf("%d files downloaded.\n", len(reqs))

}

func init() {
	var manifest string
	var downloadPath string
	var protocol string
	var batch bool
	var numParallel int

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

			protocolText := ""
			if protocol != "" {
				protocolText = "?protocol=" + protocol
			}

			var objects []ManifestObject
			manifestBytes, err := ioutil.ReadFile(manifest)
			if err != nil {
				log.Fatalf("Failed reading manifest %s, %v\n", manifest, err)
			}
			json.Unmarshal(manifestBytes, &objects)

			if batch {
				reqs := make([]*grab.Request, 0)
				for _, object := range objects {
					endPointPostfix := "/user/data/download/" + object.ObjectID + protocolText
					respURL, err := function.DoRequestWithSignedHeader(profile, "", endPointPostfix)

					if err != nil {
						if strings.Contains(err.Error(), "The provided guid") {
							log.Printf("Download error: %s\n", err)
						} else {
							log.Fatalf("Fatal download error: %s\n", err)
						}
					} else {
						req, _ := grab.NewRequest(downloadPath+"/"+object.ObjectID, respURL)

						if strings.Contains(respURL, "X-Amz-Signature") {
							req.NoResume = true
						}
						reqs = append(reqs, req)

					}
				}
				batchDownload(numParallel, reqs)
			} else {
				for _, object := range objects {
					endPointPostfix := "/user/data/download/" + object.ObjectID + protocolText
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
			}

		},
	}

	downloadManifestCmd.Flags().StringVar(&manifest, "manifest", "", "The manifest file to read from")
	downloadManifestCmd.MarkFlagRequired("manifest")
	downloadManifestCmd.Flags().StringVar(&downloadPath, "download-path", "", "The directory in which to store the downloaded files")
	downloadManifestCmd.MarkFlagRequired("download-path")
	downloadManifestCmd.Flags().StringVar(&protocol, "protocol", "", "Specify the preferred protocol with --protocol=s3")
	downloadManifestCmd.Flags().BoolVar(&batch, "batch", true, "Download in parallel")
	downloadManifestCmd.Flags().IntVar(&numParallel, "numparallel", 3, "Number of downloads to run in parallel")
	RootCmd.AddCommand(downloadManifestCmd)
}
