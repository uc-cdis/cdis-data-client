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
	"github.com/uc-cdis/gen3-client/gen3-client/logs"
	pb "gopkg.in/cheggaaa/pb.v1"

	"github.com/spf13/cobra"
)

// mockgen -destination=../mocks/mock_gen3interface.go -package=mocks . Gen3Interface

func askGen3ForFileInfo(gen3Interface Gen3Interface, profile string, guid string, protocol string, downloadPath string, filenameFormat string, rename bool, renamedFiles *[]RenamedOrSkippedFileInfo) (string, int64) {
	var fileName string
	var fileSize int64

	// If the commons has the newer Shepherd API deployed, get the filename and file size from the Shepherd API.
	// Otherwise, fall back on Indexd and Fence.
	hasShepherd, err := gen3Interface.CheckForShepherdAPI(profile)
	if err != nil {
		log.Println("Error occurred when checking for Shepherd API: " + err.Error())
		log.Println("Falling back to Indexd...")
	}
	if hasShepherd {
		endPointPostfix := commonUtils.ShepherdEndpoint + "/objects/" + guid
		_, res, err := gen3Interface.GetResponse(profile, "", endPointPostfix, "GET", "", nil)
		if err != nil {
			log.Println("Error occurred when querying filename from Shepherd: " + err.Error())
			log.Println("Using GUID for filename instead.")
			if filenameFormat != "guid" {
				*renamedFiles = append(*renamedFiles, RenamedOrSkippedFileInfo{GUID: guid, OldFilename: "N/A", NewFilename: guid})
			}
			return guid, 0
		}

		decoded := struct {
			Record struct {
				FileName string `json:"file_name"`
				Size     int64  `json:"size"`
			}
		}{}
		err = json.NewDecoder(res.Body).Decode(&decoded)
		if err != nil {
			log.Println("Error occurred when reading response from Shepherd: " + err.Error())
			log.Println("Using GUID for filename instead.")
			if filenameFormat != "guid" {
				*renamedFiles = append(*renamedFiles, RenamedOrSkippedFileInfo{GUID: guid, OldFilename: "N/A", NewFilename: guid})
			}
			return guid, 0
		}

		fileName = decoded.Record.FileName
		fileSize = decoded.Record.Size

	} else {
		// Attempt to get the filename from Indexd
		endPointPostfix := commonUtils.IndexdIndexEndpoint + "/" + guid
		indexdMsg, err := gen3Interface.DoRequestWithSignedHeader(profile, "", endPointPostfix, "", nil)
		if err != nil {
			log.Println("Error occurred when querying filename from IndexD: " + err.Error())
			log.Println("Using GUID for filename instead.")
			if filenameFormat != "guid" {
				*renamedFiles = append(*renamedFiles, RenamedOrSkippedFileInfo{GUID: guid, OldFilename: "N/A", NewFilename: guid})
			}
			return guid, 0
		}

		if filenameFormat == "guid" {
			return guid, indexdMsg.Size
		}

		actualFilename := indexdMsg.FileName
		if actualFilename == "" {
			if len(indexdMsg.URLs) > 0 {
				// Indexd record has no file name but does have URLs, try to guess file name from URL
				var indexdURL = indexdMsg.URLs[0]
				if protocol != "" {
					for _, url := range indexdMsg.URLs {
						if strings.HasPrefix(url, protocol) {
							indexdURL = url
						}
					}
				}

				actualFilename = guessFilenameFromURL(indexdURL)
				if actualFilename == "" {
					log.Println("Error occurred when guessing filename for object " + guid)
					log.Println("Using GUID for filename instead.")
					*renamedFiles = append(*renamedFiles, RenamedOrSkippedFileInfo{GUID: guid, OldFilename: "N/A", NewFilename: guid})
					return guid, indexdMsg.Size
				}
			} else {
				// Neither file name nor URLs exist in the Indexd record
				// Indexd record is busted for that file, just return as we are renaming the file for now
				// The download logic will handle the errors
				log.Println("Neither file name nor URLs exist in the Indexd record of " + guid)
				log.Println("The attempt of downloading file is likely to fail! Check Indexd record!")
				log.Println("Using GUID for filename instead.")
				*renamedFiles = append(*renamedFiles, RenamedOrSkippedFileInfo{GUID: guid, OldFilename: "N/A", NewFilename: guid})
				return guid, indexdMsg.Size
			}
		}

		fileName = actualFilename
		fileSize = indexdMsg.Size
	}

	if filenameFormat == "original" {
		if !rename { // no renaming in original mode
			return fileName, fileSize
		}
		newFilename := processOriginalFilename(downloadPath, fileName)
		if fileName != newFilename {
			*renamedFiles = append(*renamedFiles, RenamedOrSkippedFileInfo{GUID: guid, OldFilename: fileName, NewFilename: newFilename})
		}
		return newFilename, fileSize
	}
	// filenameFormat == "combined"
	combinedFilename := guid + "_" + fileName
	return combinedFilename, fileSize
}

func guessFilenameFromURL(URL string) string {
	splittedURLWithFilename := strings.Split(URL, "/")
	actualFilename := splittedURLWithFilename[len(splittedURLWithFilename)-1]
	return actualFilename
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
		fmt.Printf("WARNING: in \"guid\" or \"combined\" mode, duplicated files under \"%s\" will be overwritten\n", downloadPath)
		if !noPrompt && !commonUtils.AskForConfirmation("Proceed?") {
			log.Println("Aborted by user")
			os.Exit(0)
		}
	} else if !rename {
		fmt.Printf("WARNING: flag \"rename\" was set to false in \"original\" mode, duplicated files under \"%s\" will be overwritten\n", downloadPath)
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
		return commonUtils.FileDownloadResponseObject{DownloadPath: downloadPath, Filename: filename, Overwrite: true} // local filesize is greater than INDEXD record, overwrite local existing
	}
	// local filesize is less than INDEXD record, try ranged download
	return commonUtils.FileDownloadResponseObject{DownloadPath: downloadPath, Filename: filename, Range: localFilesize}
}

func batchDownload(g3 Gen3Interface, batchFDRSlice []commonUtils.FileDownloadResponseObject, protocolText string, workers int, errCh chan error) int {
	bars := make([]*pb.ProgressBar, 0)
	fdrs := make([]commonUtils.FileDownloadResponseObject, 0)
	for _, fdrObject := range batchFDRSlice {
		err := GetDownloadResponse(g3, profile, &fdrObject, protocolText)
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
		file, err := os.OpenFile(fdrObject.DownloadPath+fdrObject.Filename, fileFlag, 0666)
		if err != nil {
			errCh <- errors.New("Error occurred during opening local file: " + err.Error())
			continue
		}
		defer file.Close()
		defer fdrObject.Response.Body.Close()
		bar := pb.New64(fdrObject.Response.ContentLength + fdrObject.Range).SetUnits(pb.U_BYTES).SetRefreshRate(time.Millisecond * 10).Prefix(fdrObject.Filename + " ")
		bar.Set64(fdrObject.Range)
		writer := io.MultiWriter(file, bar)
		bars = append(bars, bar)
		fdrObject.Writer = writer
		fdrs = append(fdrs, fdrObject)
	}

	fdrCh := make(chan commonUtils.FileDownloadResponseObject, len(fdrs))
	pool, err := pb.StartPool(bars...)
	if err != nil {
		errCh <- errors.New("Error occurred during initializing progress bars: " + err.Error())
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

	gen3Interface := NewGen3Interface()

	log.Printf("Total number of GUIDs: %d", len(guids))
	log.Println("Preparing file info for each file, please wait...")
	fileInfoBar := pb.New(len(guids)).SetRefreshRate(time.Millisecond * 10)
	fileInfoBar.Start()
	for _, guid := range guids {
		var fdrObject commonUtils.FileDownloadResponseObject
		filename, filesize := askGen3ForFileInfo(gen3Interface, profile, guid, protocol, downloadPath, filenameFormat, rename, &renamedFiles)
		fdrObject = commonUtils.FileDownloadResponseObject{DownloadPath: downloadPath, Filename: filename}
		if !rename {
			fdrObject = validateLocalFileStat(downloadPath, filename, filesize, skipCompleted)
		}
		fdrObject.GUID = guid
		fdrObjects = append(fdrObjects, fdrObject)
		fileInfoBar.Increment()
	}
	fileInfoBar.Finish()
	log.Println("File info prepared successfully")

	totalCompeleted := 0
	workers, _, errCh, _ := initBatchUploadChannels(numParallel, len(fdrObjects))
	batchFDRSlice := make([]commonUtils.FileDownloadResponseObject, 0)
	for _, fdrObject := range fdrObjects {
		if fdrObject.Skip {
			log.Printf("File \"%s\" (GUID: %s) has been skipped because there is a complete local copy\n", fdrObject.Filename, fdrObject.GUID)
			skippedFiles = append(skippedFiles, RenamedOrSkippedFileInfo{GUID: fdrObject.GUID, OldFilename: fdrObject.Filename})
			continue
		}

		if len(batchFDRSlice) < workers {
			batchFDRSlice = append(batchFDRSlice, fdrObject)
		} else {
			totalCompeleted += batchDownload(gen3Interface, batchFDRSlice, protocolText, workers, errCh)
			batchFDRSlice = make([]commonUtils.FileDownloadResponseObject, 0)
			batchFDRSlice = append(batchFDRSlice, fdrObject)
		}
	}
	totalCompeleted += batchDownload(gen3Interface, batchFDRSlice, protocolText, workers, errCh) // download remainders

	log.Printf("%d files downloaded.\n", totalCompeleted)

	if len(renamedFiles) > 0 {
		log.Printf("%d files have been renamed as the following:\n", len(renamedFiles))
		for _, rfi := range renamedFiles {
			log.Printf("File \"%s\" (GUID: %s) has been renamed as: %s\n", rfi.OldFilename, rfi.GUID, rfi.NewFilename)
		}
	}
	if len(skippedFiles) > 0 {
		log.Printf("%d files have been skipped\n", len(skippedFiles))
	}
	if len(errCh) > 0 {
		close(errCh)
		log.Printf("%d files have encountered an error during downloading, detailed error messages are:\n", len(errCh))
		for err := range errCh {
			log.Println(err.Error())
		}
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
			// don't initialize transmission logs for non-uploading related commands
			logs.SetToBoth()

			manifestPath, _ = commonUtils.GetAbsolutePath(manifestPath)
			manifestFile, err := os.Open(manifestPath)
			if err != nil {
				log.Fatalf("Failed to open manifest file %s, %v\n", manifestPath, err)
			}
			defer manifestFile.Close()
			manifestFileStat, err := manifestFile.Stat()
			if err != nil {
				log.Fatalf("Failed to get manifest file stats %s, %v\n", manifestPath, err)
			}
			log.Println("Reading manifest...")
			manifestFileSize := manifestFileStat.Size()
			manifestFileBar := pb.New(int(manifestFileSize)).SetUnits(pb.U_BYTES).SetRefreshRate(time.Millisecond * 10)
			manifestFileBar.Start()

			manifestFileReader := manifestFileBar.NewProxyReader(manifestFile)

			manifestBytes, err := ioutil.ReadAll(manifestFileReader)
			manifestFileBar.Finish()

			if err != nil {
				log.Printf("Failed reading manifest %s, %v\n", manifestPath, err)
				log.Fatalln("A valid manifest can be acquired by using the \"Download Manifest\" button in Data Explorer from a data common's portal")
			}
			var objects []ManifestObject
			err = json.Unmarshal(manifestBytes, &objects)
			if err != nil {
				log.Fatalf("Error has occurred during unmarshalling manifest object: %v\n", err)
			}

			guids := make([]string, 0)
			for _, object := range objects {
				if object.ObjectID != "" {
					guids = append(guids, object.ObjectID)
				} else {
					log.Println("Download error: empty object_id (GUID)")
				}
			}
			downloadFile(guids, downloadPath, filenameFormat, rename, noPrompt, protocol, numParallel, skipCompleted)
			logs.CloseMessageLog()
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
