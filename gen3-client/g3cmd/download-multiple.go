package g3cmd

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/uc-cdis/gen3-client/gen3-client/commonUtils"

	"github.com/cavaliercoder/grab"
	"github.com/spf13/cobra"

	"github.com/uc-cdis/gen3-client/gen3-client/jwt"
)

func processOriginalFilename(downloadPath string, actualFilename string) string {
	_, err := os.Stat(downloadPath + actualFilename)
	if os.IsNotExist(err) {
		return actualFilename
	}
	extension := filepath.Ext(actualFilename)
	filename := strings.TrimSuffix(actualFilename, extension)
	counter := 2
	for {
		newFilename := filename + "_" + strconv.Itoa(counter) + extension
		_, err := os.Stat(downloadPath + newFilename)
		if os.IsNotExist(err) {
			return newFilename
		}
		counter++
	}
}

func processS3URLForFilename(presignedURL string, guid string, downloadPath string, filenameFormat string, overwrite bool, renamedFiles *[]RenamedFileInfo) string {
	if filenameFormat == "guid" {
		return guid
	}
	urlWithFilename := strings.Split(presignedURL, "?")[0]
	splittedFilename := strings.Split(urlWithFilename, guid+"/")
	actualFilename := splittedFilename[len(splittedFilename)-1]
	if filenameFormat == "original" {
		if overwrite {
			return actualFilename
		}
		newFilename := processOriginalFilename(downloadPath, actualFilename)
		if actualFilename != newFilename {
			*renamedFiles = append(*renamedFiles, RenamedFileInfo{GUID: guid, OldFilename: actualFilename, NewFilename: newFilename})
		}
		return newFilename
	}
	// filenameFormat == "combined"
	return guid + "_" + actualFilename
}

func validateFilenameFormat(downloadPath string, filenameFormat string, overwrite bool, noPrompt bool) {
	if filenameFormat != "original" && filenameFormat != "guid" && filenameFormat != "combined" {
		log.Fatalln("Invalid option found! Option \"filename-format\" can either be \"original\", \"guid\" or \"combined\" only")
	}
	if filenameFormat == "guid" || filenameFormat == "combined" {
		fmt.Printf("WARNING: in \"guid\" or \"combined\" mode, duplicated files under \"%s\" will be overwritten!\n", downloadPath)
		if !noPrompt && !commonUtils.AskForConfirmation("Proceed?") {
			log.Println("Aborted by user")
			os.Exit(0)
		}
	} else if overwrite {
		fmt.Printf("WARNING: flag \"overwrite\" was set to true in \"original\" mode, duplicated files under \"%s\" will be overwritten!\n", downloadPath)
		if !noPrompt && !commonUtils.AskForConfirmation("Proceed?") {
			log.Println("Aborted by user")
			os.Exit(0)
		}
	} else {
		fmt.Printf("NOTICE: flag \"overwrite\" was set to false in \"original\" mode, duplicated files under \"%s\" will be renamed by appending a counter value to the original filenames!\n", downloadPath)
	}
}

func downloadFile(guids []string, downloadPath string, filenameFormat string, overwrite bool, protocol string, numParallel int) {
	request := new(jwt.Request)
	configure := new(jwt.Configure)
	function := new(jwt.Functions)

	function.Config = configure
	function.Request = request

	protocolText := ""
	if protocol != "" {
		protocolText = "?protocol=" + protocol
	}

	err := os.MkdirAll(downloadPath, 0766)
	if err != nil {
		log.Fatal("Cannot create folder \"" + downloadPath + "\"")
	}

	renamedFiles := make([]RenamedFileInfo, 0)

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
			if strings.Contains(msg.URL, "X-Amz-Signature") { // S3 presigned URL
				filename = processS3URLForFilename(msg.URL, guid, downloadPath, filenameFormat, overwrite, &renamedFiles)
			} else {
				fmt.Println("WARNING: cannot parse URL to get actually filename, will use GUID as its filename by default.")
			}
			req, _ := grab.NewRequest(downloadPath+filename, msg.URL)
			// NoResume specifies that a partially completed download will be restarted without attempting to resume any existing file
			req.NoResume = true
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

	if len(renamedFiles) > 0 {
		fmt.Printf("\n%d files have been renamed as the following:\n", len(renamedFiles))
		for _, rfi := range renamedFiles {
			fmt.Printf("File \"%s\" (GUID %s) has been renamed as: %s\n", rfi.OldFilename, rfi.GUID, rfi.NewFilename)
		}
	}
}

func init() {
	var manifestPath string
	var downloadPath string
	var filenameFormat string
	var overwrite bool
	var noPrompt bool
	var protocol string
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

			downloadPath = commonUtils.ParseRootPath(downloadPath)
			if !strings.HasSuffix(downloadPath, "/") {
				downloadPath += "/"
			}
			filenameFormat = strings.ToLower(strings.TrimSpace(filenameFormat))
			validateFilenameFormat(downloadPath, filenameFormat, overwrite, noPrompt)

			var objects []ManifestObject
			manifestBytes, err := ioutil.ReadFile(manifestPath)
			if err != nil {
				log.Printf("Failed reading manifest %s, %v\n", manifestPath, err)
				log.Fatalln("A valid manifest can be acquired by using the \"Download Manifest\" button on " + dataExplorerURL)
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
			downloadFile(guids, downloadPath, filenameFormat, overwrite, protocol, numParallel)
		},
	}

	downloadMultipleCmd.Flags().StringVar(&manifestPath, "manifest", "", "The manifest file to read from")
	downloadMultipleCmd.Flags().StringVar(&downloadPath, "download-path", ".", "The directory in which to store the downloaded files")
	downloadMultipleCmd.Flags().StringVar(&filenameFormat, "filename-format", "original", "The format of filename to be used, including \"original\", \"guid\" and \"combined\" (default: original)")
	downloadMultipleCmd.Flags().BoolVar(&overwrite, "overwrite", false, "Only useful when \"--filename-format=original\", will overwrite any duplicates in \"download-path\" if set to true, will rename file by appending a counter value to its filename otherwise (default: false)")
	downloadMultipleCmd.Flags().BoolVar(&noPrompt, "no-prompt", false, "If set to true, will not display user prompt message for confirmation (default: false)")
	downloadMultipleCmd.Flags().StringVar(&protocol, "protocol", "", "Specify the preferred protocol with --protocol=s3 (default: \"\")")
	downloadMultipleCmd.Flags().IntVar(&numParallel, "numparallel", 1, "Number of downloads to run in parallel (default: 1)")
	RootCmd.AddCommand(downloadMultipleCmd)
}
