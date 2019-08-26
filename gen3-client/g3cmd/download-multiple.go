package g3cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/uc-cdis/gen3-client/gen3-client/commonUtils"
	pb "gopkg.in/cheggaaa/pb.v1"

	"github.com/spf13/cobra"

	"github.com/uc-cdis/gen3-client/gen3-client/jwt"
)

func askIndexDForFileInfo(guid string, downloadPath string, filenameFormat string, rename bool, renamedFiles *[]RenamedOrSkippedFileInfo) (string, int64) {
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
		if filenameFormat != "guid" {
			*renamedFiles = append(*renamedFiles, RenamedOrSkippedFileInfo{GUID: guid, OldFilename: "N/A", NewFilename: guid})
		}
		return guid, 0
	}

	if filenameFormat == "guid" {
		return guid, msg.Size
	}

	actualFilename := msg.FileName

	if filenameFormat == "original" {
		if !rename {
			return actualFilename, msg.Size
		}
		newFilename := processOriginalFilename(downloadPath, actualFilename)
		if actualFilename != newFilename {
			*renamedFiles = append(*renamedFiles, RenamedOrSkippedFileInfo{GUID: guid, OldFilename: actualFilename, NewFilename: newFilename})
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

func validateFilenameFormat(downloadPath string, filenameFormat string, rename bool, noPrompt bool) {
	if filenameFormat != "original" && filenameFormat != "guid" && filenameFormat != "combined" {
		log.Fatalln("Invalid option found! Option \"filename-format\" can either be \"original\", \"guid\" or \"combined\" only")
	}
	if filenameFormat == "guid" || filenameFormat == "combined" {
		fmt.Printf("WARNING: in \"guid\" or \"combined\" mode, duplicated files under \"%s\" may be overwritten\n", downloadPath)
		if !noPrompt && !commonUtils.AskForConfirmation("Proceed?") {
			log.Println("Aborted by user")
			os.Exit(0)
		}
	} else if !rename {
		fmt.Printf("WARNING: flag \"rename\" was set to false in \"original\" mode, duplicated files under \"%s\" may be overwritten\n", downloadPath)
		if !noPrompt && !commonUtils.AskForConfirmation("Proceed?") {
			log.Println("Aborted by user")
			os.Exit(0)
		}
	} else {
		fmt.Printf("NOTICE: flag \"rename\" was set to true in \"original\" mode, duplicated files under \"%s\" will be renamed by appending a counter value to the original filenames\n", downloadPath)
	}
}

func validateLocalFileStat(downloadPath string, filename string, filesize int64, skipCompleted bool) commonUtils.FileDownloadResponseObject {
	fi, err := os.Stat(downloadPath + filename) // check filename for local existence
	if err != nil {
		if os.IsNotExist(err) {
			return commonUtils.FileDownloadResponseObject{DownloadPath: downloadPath, Filename: filename} // no local file, normal full length download
		}
		log.Printf("Error occurred when getting information for file \"%s\": %s\n", downloadPath+filename, err.Error())
		log.Println("Will try to download the whole file")
		return commonUtils.FileDownloadResponseObject{DownloadPath: downloadPath, Filename: filename} // errorred when trying to get local FI, normal full length download
	}

	// have existing local file and may want to skip, check more conditions
	if !skipCompleted {
		return commonUtils.FileDownloadResponseObject{DownloadPath: downloadPath, Filename: filename, Overwrite: true} // not skipping any local files, normal full length download
	}

	localFilesize := fi.Size()
	if localFilesize == filesize {
		return commonUtils.FileDownloadResponseObject{DownloadPath: downloadPath, Filename: filename, Skip: true} // both filename and filesize matches, consider as completed
	}
	if localFilesize > filesize {
		return commonUtils.FileDownloadResponseObject{DownloadPath: downloadPath, Filename: filename, Overwrite: true} // local filesize greater than INDEXD record, overwrite local existing
	}
	// local filesize greater less than INDEXD record, try ranged download
	return commonUtils.FileDownloadResponseObject{DownloadPath: downloadPath, Filename: filename, Range: filesize - localFilesize}
}

func batchDownload(batchFDRSlice []commonUtils.FileDownloadResponseObject, protocolText string, workers int, errCh chan error) int {
	bars := make([]*pb.ProgressBar, 0)
	fdrs := make([]commonUtils.FileDownloadResponseObject, 0)
	for _, fdrObject := range batchFDRSlice {
		err := GetDownloadResponse(&fdrObject, protocolText)
		if err != nil {
			errCh <- err
			continue
		}

		fileFlag := os.O_CREATE | os.O_RDWR
		if fdrObject.Range != 0 {
			fileFlag = os.O_APPEND | os.O_RDWR
		} else if fdrObject.Overwrite {
			fileFlag = os.O_TRUNC | os.O_RDWR
		}

		subDir := filepath.Dir(fdrObject.Filename)
		if subDir != "." && subDir != "/" {
			os.MkdirAll(fdrObject.DownloadPath+subDir, 0766)
		}
		file, err := os.OpenFile(fdrObject.DownloadPath+fdrObject.Filename, fileFlag, 0766)
		if err != nil {
			errCh <- errors.New("Error occurred during opening local file: " + err.Error())
			continue
		}
		defer file.Close()
		defer fdrObject.Response.Body.Close()
		bar := pb.New64(fdrObject.Response.ContentLength).SetUnits(pb.U_BYTES).SetRefreshRate(time.Millisecond * 10).Prefix(fdrObject.Filename + " ")
		writer := io.MultiWriter(file, bar)
		bars = append(bars, bar)
		fdrObject.Writer = writer
		fdrs = append(fdrs, fdrObject)
	}

	fdrCh := make(chan commonUtils.FileDownloadResponseObject, len(fdrs))
	pool, err := pb.StartPool(bars...)
	if err != nil {
		errCh <- errors.New("Error occurred during starting progress bar pool: " + err.Error())
		return 0
	}

	wg := sync.WaitGroup{}
	succeeded := 0
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			for fdr := range fdrCh {
				if _, err = io.Copy(fdr.Writer, fdr.Response.Body); err != nil {
					errCh <- errors.New("io.Copy error: " + err.Error())
					return
				}
				succeeded++
			}
			wg.Done()
		}()
	}

	for _, fdr := range fdrs {
		fdrCh <- fdr
	}
	close(fdrCh)

	wg.Wait()
	pool.Stop()
	return succeeded
}

func downloadFile(guids []string, downloadPath string, filenameFormat string, rename bool, noPrompt bool, protocol string, numParallel int, skipCompleted bool) {
	if numParallel < 1 {
		log.Fatalln("Invalid value for option \"numparallel\": must be a positive integer! Please check your input.")
	}

	downloadPath = commonUtils.ParseRootPath(downloadPath)
	if !strings.HasSuffix(downloadPath, "/") {
		downloadPath += "/"
	}
	filenameFormat = strings.ToLower(strings.TrimSpace(filenameFormat))
	if (filenameFormat == "guid" || filenameFormat == "combined") && rename {
		fmt.Println("NOTICE: flag \"rename\" only works if flag \"filename-format\" is \"original\"")
		rename = false
	}
	validateFilenameFormat(downloadPath, filenameFormat, rename, noPrompt)

	protocolText := ""
	if protocol != "" {
		protocolText = "?protocol=" + protocol
	}

	err := os.MkdirAll(downloadPath, 0766)
	if err != nil {
		log.Fatalln("Cannot create folder \"" + downloadPath + "\"")
	}

	renamedFiles := make([]RenamedOrSkippedFileInfo, 0)
	skippedFiles := make([]RenamedOrSkippedFileInfo, 0)
	fdrObjects := make([]commonUtils.FileDownloadResponseObject, 0)

	for _, guid := range guids {
		var fdrObject commonUtils.FileDownloadResponseObject
		filename, filesize := askIndexDForFileInfo(guid, downloadPath, filenameFormat, rename, &renamedFiles)
		fdrObject = commonUtils.FileDownloadResponseObject{DownloadPath: downloadPath, Filename: filename}
		if !rename {
			fdrObject = validateLocalFileStat(downloadPath, filename, filesize, skipCompleted)
		}
		fdrObject.GUID = guid
		fdrObjects = append(fdrObjects, fdrObject)
	}

	totalCompeleted := 0
	workers, _, errCh, _ := initBatchUploadChannels(numParallel, len(fdrObjects))
	batchFDRSlice := make([]commonUtils.FileDownloadResponseObject, 0)
	for i, fdrObject := range fdrObjects {
		if fdrObject.Skip {
			skippedFiles = append(skippedFiles, RenamedOrSkippedFileInfo{GUID: fdrObject.GUID, OldFilename: fdrObject.Filename})
			continue
		}

		if len(batchFDRSlice) < workers {
			batchFDRSlice = append(batchFDRSlice, fdrObject)
		} else {
			totalCompeleted += batchDownload(batchFDRSlice, protocolText, workers, errCh)
			batchFDRSlice = make([]commonUtils.FileDownloadResponseObject, 0)
			batchFDRSlice = append(batchFDRSlice, fdrObject)
		}
		if i == len(fdrObjects)-1 { // download remainders
			totalCompeleted += batchDownload(batchFDRSlice, protocolText, workers, errCh)
		}
	}

	fmt.Printf("%d files downloaded.\n", totalCompeleted)

	if len(renamedFiles) > 0 {
		fmt.Printf("\n%d files have been renamed as the following:\n", len(renamedFiles))
		for _, rfi := range renamedFiles {
			fmt.Printf("File \"%s\" (GUID %s) has been renamed as: %s\n", rfi.OldFilename, rfi.GUID, rfi.NewFilename)
		}
	}
	if len(skippedFiles) > 0 {
		fmt.Printf("\n%d files have been skipped:\n", len(skippedFiles))
		for _, sfi := range skippedFiles {
			fmt.Printf("File \"%s\" (GUID %s) has been skipped\n", sfi.OldFilename, sfi.GUID)
		}
	}
	if len(errCh) > 0 {
		fmt.Printf("\n%d files have errorred during downloading\n", len(errCh))
	}
}

func init() {
	var manifestPath string
	var downloadPath string
	var filenameFormat string
	var rename bool
	var noPrompt bool
	var protocol string
	var numParallel int
	var skipCompleted bool

	var downloadMultipleCmd = &cobra.Command{
		Use:     "download-multiple",
		Short:   "Download multiple of files from a specified manifest",
		Long:    `Get presigned URLs for multiple of files specified in a manifest file and then download all of them.`,
		Example: `./gen3-client download-multiple --profile=<profile-name> --manifest=<path-to-manifest/manifest.json> --download-path=<path-to-file-dir/>`,
		Run: func(cmd *cobra.Command, args []string) {

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
			downloadFile(guids, downloadPath, filenameFormat, rename, noPrompt, protocol, numParallel, skipCompleted)
		},
	}

	downloadMultipleCmd.Flags().StringVar(&profile, "profile", "", "Specify profile to use")
	downloadMultipleCmd.MarkFlagRequired("profile")
	downloadMultipleCmd.Flags().StringVar(&manifestPath, "manifest", "", "The manifest file to read from. A valid manifest can be acquired by using the \"Download Manifest\" button in Data Explorer from a data common's portal")
	downloadMultipleCmd.MarkFlagRequired("manifest")
	downloadMultipleCmd.Flags().StringVar(&downloadPath, "download-path", ".", "The directory in which to store the downloaded files")
	downloadMultipleCmd.Flags().StringVar(&filenameFormat, "filename-format", "original", "The format of filename to be used, including \"original\", \"guid\" and \"combined\"")
	downloadMultipleCmd.Flags().BoolVar(&rename, "rename", false, "Only useful when \"--filename-format=original\", will rename file by appending a counter value to its filename if set to true, otherwise the same filename will be used")
	downloadMultipleCmd.Flags().BoolVar(&noPrompt, "no-prompt", false, "If set to true, will not display user prompt message for confirmation")
	downloadMultipleCmd.Flags().StringVar(&protocol, "protocol", "", "Specify the preferred protocol with --protocol=s3")
	downloadMultipleCmd.Flags().IntVar(&numParallel, "numparallel", 1, "Number of downloads to run in parallel")
	downloadMultipleCmd.Flags().BoolVar(&skipCompleted, "skip-completed", false, "If set to true, will check for filename and size before download and skip any files in \"download-path\" that matches both")
	RootCmd.AddCommand(downloadMultipleCmd)
}
