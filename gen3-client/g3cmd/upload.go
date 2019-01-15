package g3cmd

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"

	homedir "github.com/mitchellh/go-homedir"
	"github.com/spf13/cobra"
	"github.com/uc-cdis/gen3-client/gen3-client/commonUtils"
	"github.com/uc-cdis/gen3-client/gen3-client/jwt"
	pb "gopkg.in/cheggaaa/pb.v1"
)

var historyFile string
var historyFileMap map[string]string

func initHistory() {
	home, err := homedir.Dir()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	if _, err := os.Stat(home + "/.gen3/"); os.IsNotExist(err) { // path to ~/.gen3 does not exist
		err = os.Mkdir(home+"/.gen3/", 0644)
		if err != nil {
			log.Fatal("Cannot create folder \"" + home + "/.gen3/\"")
			os.Exit(1)
		}
		fmt.Println("Created folder \"" + home + "/.gen3/\"")
	}

	historyFile = home + "/.gen3/" + profile + "_history.json"

	file, _ := os.OpenFile(historyFile, os.O_RDWR|os.O_CREATE, 0644)
	fi, err := file.Stat()
	if err != nil {
		log.Fatal("Error occurred when opening file \"" + historyFile + "\": " + err.Error())
	}
	fmt.Println("Local history file \"" + historyFile + "\" has opened")

	historyFileMap = make(map[string]string)
	if fi.Size() > 0 {
		data, err := ioutil.ReadAll(file)
		if err != nil {
			log.Fatal("Error occurred when reading from file \"" + historyFile + "\": " + err.Error())
		}

		err = json.Unmarshal(data, &historyFileMap)
		if err != nil {
			log.Fatal("Error occurred when unmarshaling JSON objects: " + err.Error())
		}
	}
}

func init() {
	var uploadPath string
	var batch bool
	var numParallel int

	var uploadNewCmd = &cobra.Command{
		Use:   "upload",
		Short: "upload file(s) to object storage.",
		Long:  `Gets a presigned URL for each file and then uploads the specified file(s).`,
		Example: "For uploading a single file:\n./gen3-client upload --profile=<profile-name> --upload-path=<path-to-files/data.bam>\n" +
			"For uploading all files within an folder:\n./gen3-client upload --profile=<profile-name> --upload-path=<path-to-files/folder/>\n" +
			"Can also support regex such as:\n./gen3-client upload --profile=<profile-name> --upload-path=<path-to-files/folder/*>\n" +
			"Or:\n./gen3-client upload --profile=<profile-name> --upload-path=<path-to-files/*/folder/*.bam>",
		Run: func(cmd *cobra.Command, args []string) {
			initHistory()

			request := new(jwt.Request)
			configure := new(jwt.Configure)
			function := new(jwt.Functions)

			function.Config = configure
			function.Request = request

			filePaths, err := commonUtils.ParseFilePaths(uploadPath)
			if err != nil {
				log.Fatalf(err.Error())
			}
			if len(filePaths) == 0 {
				log.Fatalf("Error when parsing file paths, no file has been found in the provided location \"" + uploadPath + "\"")
			}
			fmt.Println("\nThe following file(s) has been found in path \"" + uploadPath + "\" and will be uploaded:")
			for _, filePath := range filePaths {
				file, _ := os.Open(filePath)
				if fi, _ := file.Stat(); !fi.IsDir() {
					fmt.Println("\t" + filePath)
				}
				file.Close()
			}
			fmt.Println()

			reqs := make([]*http.Request, 0)
			bars := make([]*pb.ProgressBar, 0)
			for _, filePath := range filePaths {
				file, err := os.Open(filePath)
				if err != nil {
					log.Fatal("File open error")
				}
				defer file.Close()

				if fi, _ := file.Stat(); fi.IsDir() {
					continue
				}

				_, present := historyFileMap[filePath]
				if present {
					fmt.Println("File \"" + filePath + "\" has been found in local submission history and has be skipped for preventing duplicated submissions.")
					continue
				}
				endPointPostfix := "/user/data/upload"
				object := NewFlowRequestObject{Filename: filepath.Base(filePath)}
				objectBytes, err := json.Marshal(object)

				respURL, guid, err := function.DoRequestWithSignedHeader(profile, "", endPointPostfix, "application/json", objectBytes)

				if respURL == "" || guid == "" {
					if err != nil {
						log.Fatalf("You don't have permission to upload data, detailed error message: " + err.Error())
					} else {
						log.Fatalf("Unknown error has occurred during presigned URL or GUID generation")
					}
				}

				if _, err := os.Stat(filePath); os.IsNotExist(err) {
					log.Fatalf("The file you specified \"%s\" does not exist locally.", filePath)
				}

				req, bar, err := GenerateUploadRequest("", respURL, file)
				if err != nil {
					log.Fatalf("Error occurred during request generation: %s", err.Error())
					continue
				}
				if batch {
					reqs = append(reqs, req)
					bars = append(bars, bar)
				} else {
					uploadFile(req, bar, guid, filePath)
					file.Close()
				}
				historyFileMap[filePath] = guid
			}

			if batch {
				batchUpload(numParallel, reqs, bars)
			}

			jsonData, err := json.Marshal(historyFileMap)
			if err != nil {
				panic(err)
			}
			jsonFile, err := os.OpenFile(historyFile, os.O_RDWR|os.O_CREATE, 0666)
			if err != nil {
				panic(err)
			}
			defer jsonFile.Close()

			jsonFile.Write(jsonData)
			jsonFile.Close()
			fmt.Println("Local history data updated in ", jsonFile.Name())
		},
	}

	uploadNewCmd.Flags().StringVar(&uploadPath, "upload-path", "", "The directory or file in which contains file(s) to be uploaded")
	uploadNewCmd.MarkFlagRequired("upload-path")
	uploadNewCmd.Flags().BoolVar(&batch, "batch", false, "Upload in parallel")
	uploadNewCmd.Flags().IntVar(&numParallel, "numparallel", 3, "Number of uploads to run in parallel")
	RootCmd.AddCommand(uploadNewCmd)
}
