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

	"github.com/cavaliercoder/grab"
	"github.com/uc-cdis/gen3-client/gen3-client/commonUtils"

	"github.com/spf13/cobra"

	"github.com/uc-cdis/gen3-client/gen3-client/jwt"
)

func askIndexDForFileInfo(guid string, downloadPath string, filenameFormat string, overwrite bool, renamedFiles *[]RenamedFileInfo) (string, int64) {
	request := new(jwt.Request)
	configure := new(jwt.Configure)
	function := new(jwt.Functions)

	function.Config = configure
	function.Request = request

	endPointPostfix := "/index/index/" + guid
	msg, err := function.DoRequestWithSignedHeader(profile, "", endPointPostfix, "", nil)
	if err != nil {
		log.Println("Error occurred when querying filename from IndexD: " + err.Error())
		log.Println("Using GUID for filename instead.")
		return guid, 0

	}

	if filenameFormat == "guid" {
		return guid, msg.Size
	}

	actualFilename := msg.FileName

	if filenameFormat == "original" {
		if overwrite {
			return actualFilename, msg.Size
		}
		newFilename := processOriginalFilename(downloadPath, actualFilename)
		if actualFilename != newFilename {
			*renamedFiles = append(*renamedFiles, RenamedFileInfo{GUID: guid, OldFilename: actualFilename, NewFilename: newFilename})
		}
		return newFilename, msg.Size
	}
	// filenameFormat == "combined"
	combinedFilename := guid + "_" + actualFilename
	return combinedFilename, msg.Size
}

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

func batchDownload(numParallel int, reqs []*grab.Request) int {
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
	return completed
}

func downloadFile(guids []string, downloadPath string, filenameFormat string, overwrite bool, protocol string, numParallel int) {
	request := new(jwt.Request)
	configure := new(jwt.Configure)
	function := new(jwt.Functions)

	function.Config = configure
	function.Request = request

	if numParallel < 1 {
		log.Fatalln("Invalid value for option \"numparallel\": must be a positive integer! Please check your input.")
	}

	protocolText := ""
	if protocol != "" {
		protocolText = "?protocol=" + protocol
	}

	err := os.MkdirAll(downloadPath, 0766)
	if err != nil {
		log.Fatalln("Cannot create folder \"" + downloadPath + "\"")
	}

	renamedFiles := make([]RenamedFileInfo, 0)

	reqs := make([]*grab.Request, 0)
	totalCompeleted := 0
	for i, guid := range guids {
		endPointPostfix := "/user/data/download/" + guid + protocolText
		msg, err := function.DoRequestWithSignedHeader(profile, "", endPointPostfix, "", nil)

		if err != nil {
			log.Printf("Download error: %s\n", err)
		} else if msg.URL == "" {
			log.Printf("Error in getting download URL for object %s\n", guid)
		} else {
			filename, _ := askIndexDForFileInfo(guid, downloadPath, filenameFormat, overwrite, &renamedFiles)
			fmt.Println("WARNING: cannot parse URL to get actually filename, will use GUID as its filename by default.")
			req, _ := grab.NewRequest(downloadPath+filename, msg.URL)
			// NoResume specifies that a partially completed download will be restarted without attempting to resume any existing file
			req.NoResume = true
			reqs = append(reqs, req)
		}

		if len(reqs) == numParallel || i == len(guids)-1 {
			totalCompeleted += batchDownload(numParallel, reqs)
			reqs = make([]*grab.Request, 0)
		}
	}

	fmt.Printf("%d files downloaded.\n", totalCompeleted)

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
				log.Fatalln("A valid manifest can be acquired by using the \"Download Manifest\" button in Data Explorer from a data common's portal")
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

	downloadMultipleCmd.Flags().StringVar(&profile, "profile", "", "Specify profile to use")
	downloadMultipleCmd.MarkFlagRequired("profile")
	downloadMultipleCmd.Flags().StringVar(&manifestPath, "manifest", "", "The manifest file to read from. A valid manifest can be acquired by using the \"Download Manifest\" button in Data Explorer from a data common's portal")
	downloadMultipleCmd.MarkFlagRequired("manifest")
	downloadMultipleCmd.Flags().StringVar(&downloadPath, "download-path", ".", "The directory in which to store the downloaded files")
	downloadMultipleCmd.Flags().StringVar(&filenameFormat, "filename-format", "original", "The format of filename to be used, including \"original\", \"guid\" and \"combined\"")
	downloadMultipleCmd.Flags().BoolVar(&overwrite, "overwrite", false, "Only useful when \"--filename-format=original\", will overwrite any duplicates in \"download-path\" if set to true, will rename file by appending a counter value to its filename otherwise (default: false)")
	downloadMultipleCmd.Flags().BoolVar(&noPrompt, "no-prompt", false, "If set to true, will not display user prompt message for confirmation (default: false)")
	downloadMultipleCmd.Flags().StringVar(&protocol, "protocol", "", "Specify the preferred protocol with --protocol=s3 (default: \"\")")
	downloadMultipleCmd.Flags().IntVar(&numParallel, "numparallel", 1, "Number of downloads to run in parallel")
	RootCmd.AddCommand(downloadMultipleCmd)
}
