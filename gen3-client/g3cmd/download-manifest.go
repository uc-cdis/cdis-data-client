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

func lastString(ss []string) string {
	return ss[len(ss)-1]
}

func processS3URLForFilename(presignedURL string, guid string, filenameFormat string) string {
	if filenameFormat != "guid" {
		urlWithFilename := strings.Split(presignedURL, "?")[0]
		actualFilename := lastString(strings.Split(urlWithFilename, guid+"/"))
		if actualFilename != "" {
			if filenameFormat == "original" {
				return actualFilename
			} else if filenameFormat == "combined" {
				return guid + "." + actualFilename
			}
		}
	}
	return guid
}

func downloadFile(guids []string, downloadPath string, filenameFormat string, protocol string, numParallel int) {
	request := new(jwt.Request)
	configure := new(jwt.Configure)
	function := new(jwt.Functions)

	function.Config = configure
	function.Request = request

	protocolText := ""
	if protocol != "" {
		protocolText = "?protocol=" + protocol
	}

	downloadPath = commonUtils.ParseRootPath(downloadPath)
	if !strings.HasSuffix(downloadPath, "/") {
		downloadPath += "/"
	}

	reqs := make([]*grab.Request, 0)
	for _, guid := range guids {
		endPointPostfix := "/user/data/download/" + guid + protocolText
		msg, err := function.DoRequestWithSignedHeader(profile, "", endPointPostfix, "", nil)

		if err != nil {
			log.Printf("Download error: %s\n", err)
		} else if msg.URL == "" {
			log.Printf("Error in getting download URL for object %s\n", guid)
		} else {
			filename := guid
			if strings.Contains(msg.URL, "X-Amz-Signature") {
				filename = processS3URLForFilename(msg.URL, guid, filenameFormat)
			}
			req, _ := grab.NewRequest(downloadPath+filename, msg.URL)
			if strings.Contains(msg.URL, "X-Amz-Signature") {
				req.NoResume = true
			}
			reqs = append(reqs, req)
		}
	}

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
	var filenameFormat string
	var protocol string
	var numParallel int

	var downloadManifestCmd = &cobra.Command{
		Use:     "download-manifest",
		Short:   "Download files from a specified manifest",
		Long:    `Gets a presigned URL for a file from a GUID and then downloads the specified file.`,
		Example: `./gen3-client download-manifest --profile=<profile-name> --manifest=<path-to-manifest/manifest.json> --download-path=<path-to-file-dir/>`,
		Run: func(cmd *cobra.Command, args []string) {
			filenameFormat = strings.ToLower(strings.TrimSpace(filenameFormat))
			if filenameFormat != "original" && filenameFormat != "guid" && filenameFormat != "combined" {
				log.Fatalln("Invalid option found! Option \"filename-format\" can either be \"original\", \"guid\" or \"combined\" only")
			}

			var objects []ManifestObject
			manifestBytes, err := ioutil.ReadFile(manifest)
			if err != nil {
				log.Fatalf("Failed reading manifest %s, %v\n", manifest, err)
			}
			json.Unmarshal(manifestBytes, &objects)

			guids := make([]string, 0)
			for _, object := range objects {
				if object.ObjectID != "" {
					guids = append(guids, object.ObjectID)
				} else {
					log.Println("Download error: empty object_id (GUID)")
				}
			}
			downloadFile(guids, downloadPath, filenameFormat, protocol, numParallel)
		},
	}

	downloadManifestCmd.Flags().StringVar(&manifest, "manifest", "", "The manifest file to read from")
	downloadManifestCmd.MarkFlagRequired("manifest")
	downloadManifestCmd.Flags().StringVar(&downloadPath, "download-path", ".", "The directory in which to store the downloaded files")
	downloadManifestCmd.Flags().StringVar(&filenameFormat, "filename-format", "original", "format of filename to be used, including \"original\", \"guid\" and \"combined\"")
	downloadManifestCmd.Flags().StringVar(&protocol, "protocol", "", "Specify the preferred protocol with --protocol=s3")
	downloadManifestCmd.Flags().IntVar(&numParallel, "numparallel", 1, "Number of downloads to run in parallel")
	RootCmd.AddCommand(downloadManifestCmd)
}
