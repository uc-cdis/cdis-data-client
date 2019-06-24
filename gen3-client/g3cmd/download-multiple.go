package g3cmd

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"time"

	"github.com/uc-cdis/gen3-client/gen3-client/commonUtils"

	"github.com/cavaliercoder/grab"
	"github.com/spf13/cobra"

	"github.com/uc-cdis/gen3-client/gen3-client/jwt"
)

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
	var manifestPath string
	var downloadPath string
	var protocol string
	var batch bool
	var numParallel int

	var downloadMultipleCmd = &cobra.Command{
		Use:     "download-multiple",
		Short:   "Download multiple of files from a specified manifest",
		Long:    `Get presigned URLs for multiple of files specified in a manifest file and then download all of them.`,
		Example: `./gen3-client download-multiple --profile=<profile-name> --manifest=<path-to-manifest/manifest.json> --download-path=<path-to-file-dir/>`,
		Run: func(cmd *cobra.Command, args []string) {

			request := new(jwt.Request)
			configure := new(jwt.Configure)
			function := new(jwt.Functions)

			function.Config = configure
			function.Request = request

			host, err := function.GetHost(profile, "")
			if err != nil {
				log.Fatalln("Error occurred during parsing config file for hostname: " + err.Error())
			}
			dataExplorerURL := host.Scheme + "://" + host.Host + "/explorer"
			if manifestPath == "" {
				log.Println("Required flag \"manifest\" not set")
				log.Fatalln("A valid manifest can be acquired by using the \"Download Manifest\" button on " + dataExplorerURL)
			}

			protocolText := ""
			if protocol != "" {
				protocolText = "?protocol=" + protocol
			}

			downloadPath = commonUtils.ParseRootPath(downloadPath)

			var objects []ManifestObject
			manifestBytes, err := ioutil.ReadFile(manifestPath)
			if err != nil {
				log.Printf("Failed reading manifest %s, %v\n", manifestPath, err)
				log.Fatalln("A valid manifest can be acquired by using the \"Download Manifest\" button on " + dataExplorerURL)
			}
			json.Unmarshal(manifestBytes, &objects)

			if batch {
				reqs := make([]*grab.Request, 0)
				for _, object := range objects {
					if object.ObjectID != "" {
						endPointPostfix := "/user/data/download/" + object.ObjectID + protocolText
						msg, err := function.DoRequestWithSignedHeader(profile, "", endPointPostfix, "", nil)

						if err != nil {
							log.Printf("Download error: %s\n", err)
						} else if msg.URL == "" {
							log.Printf("Error in getting download URL for object %s\n", object.ObjectID)
						} else {
							req, _ := grab.NewRequest(downloadPath+"/"+object.ObjectID, msg.URL)
							if strings.Contains(msg.URL, "X-Amz-Signature") {
								req.NoResume = true
							}
							reqs = append(reqs, req)
						}
					} else {
						log.Println("Download error: empty object_id (GUID)")
					}
				}
				batchDownload(numParallel, reqs)
			} else {
				for _, object := range objects {
					if object.ObjectID != "" {
						endPointPostfix := "/user/data/download/" + object.ObjectID + protocolText
						msg, err := function.DoRequestWithSignedHeader(profile, "", endPointPostfix, "", nil)

						if err != nil {
							log.Printf("Download error: %s\n", err)
						} else if msg.URL == "" {
							log.Printf("Error in getting download URL for object %s\n", object.ObjectID)
						} else {
							downloadFile(object.ObjectID, downloadPath+"/"+object.ObjectID, msg.URL)
						}
					} else {
						log.Println("Download error: empty object_id (GUID)")
					}
				}
			}

		},
	}

	downloadMultipleCmd.Flags().StringVar(&manifestPath, "manifest", "", "The manifest file to read from")
	downloadMultipleCmd.Flags().StringVar(&downloadPath, "download-path", "", "The directory in which to store the downloaded files")
	downloadMultipleCmd.MarkFlagRequired("download-path")
	downloadMultipleCmd.Flags().StringVar(&protocol, "protocol", "", "Specify the preferred protocol with --protocol=s3 (default: \"\")")
	downloadMultipleCmd.Flags().BoolVar(&batch, "batch", true, "Download in parallel (default: true)")
	downloadMultipleCmd.Flags().IntVar(&numParallel, "numparallel", 3, "Number of downloads to run in parallel (default: 3)")
	RootCmd.AddCommand(downloadMultipleCmd)
}
