package g3cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	homedir "github.com/mitchellh/go-homedir"
	"github.com/spf13/cobra"
	"github.com/uc-cdis/gen3-client/gen3-client/commonUtils"
	"github.com/uc-cdis/gen3-client/gen3-client/jwt"
	pb "gopkg.in/cheggaaa/pb.v1"
)

var historyFile string
var historyFileMap map[string]string
var pathSeparator = string(os.PathSeparator)

type fileInfo struct {
	filepath string
	filename string
}

func initHistory() {
	home, err := homedir.Dir()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	historyPath := home + pathSeparator + ".gen3" + pathSeparator

	if _, err := os.Stat(historyPath); os.IsNotExist(err) { // path to ~/.gen3 does not exist
		err = os.Mkdir(historyPath, 0644)
		if err != nil {
			log.Fatal("Cannot create folder \"" + historyPath + "\"")
			os.Exit(1)
		}
		fmt.Println("Created folder \"" + historyPath + "\"")
	}

	historyFile = historyPath + profile + "_history.json"

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

func validateFilePath(filePaths []string) []string {
	validatedFilePaths := make([]string, 0)
	for _, filePath := range filePaths {
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			log.Printf("The file you specified \"%s\" does not exist locally", filePath)
			continue
		}

		file, err := os.Open(filePath)
		if err != nil {
			log.Printf("File open error")
			continue
		}

		if fi, _ := file.Stat(); fi.IsDir() {
			continue
		}

		_, present := historyFileMap[filePath]
		if present {
			fmt.Println("File \"" + filePath + "\" has been found in local submission history and has be skipped for preventing duplicated submissions.")
			continue
		}
		validatedFilePaths = append(validatedFilePaths, filePath)
		file.Close()
	}
	return validatedFilePaths
}

func processFilename(uploadPath string, filePath string, includeSubDirName bool) (fileInfo, error) {
	var err error
	filename := filepath.Base(filePath)
	if includeSubDirName {
		presentDirname := strings.TrimSuffix(commonUtils.ParseRootPath(uploadPath), pathSeparator+"*")
		subFilename := strings.TrimPrefix(filePath, presentDirname)
		dir, _ := filepath.Split(subFilename)
		if dir != "" && dir != string(pathSeparator) {
			filename = strings.TrimPrefix(subFilename, pathSeparator)
			filename = strings.Replace(filename, pathSeparator, ".", -1)
		} else {
			err = errors.New("Include subdirectory names will only works if the file is under at least one subdirectory.")
		}
	}
	return fileInfo{filePath, filename}, err
}

func getPresignedURL(function *jwt.Functions, filePath string, uploadPath string, includeSubDirName bool) (string, string, error) {
	fileinfo, err := processFilename(uploadPath, filePath, includeSubDirName)
	if err != nil {
		fmt.Println(err.Error())
	}
	endPointPostfix := "/user/data/upload"
	object := NewFlowRequestObject{Filename: fileinfo.filename}
	objectBytes, err := json.Marshal(object)

	respURL, guid, err := function.DoRequestWithSignedHeader(profile, "", endPointPostfix, "application/json", objectBytes)

	if respURL == "" || guid == "" {
		if err != nil {
			return "", "", errors.New("You don't have permission to upload data, detailed error message: " + err.Error())
		}
		return "", "", errors.New("Unknown error has occurred during presigned URL or GUID generation. Please check logs from Gen3 services")
	}
	return respURL, guid, err
}

func init() {
	var uploadPath string
	var batch bool
	var numParallel int
	var includeSubDirName bool

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

			uploadPath = filepath.Clean(uploadPath)
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

			validatedFilePaths := validateFilePath(filePaths)

			if batch {
				workers := numParallel
				if workers < 1 || workers > len(validatedFilePaths) {
					workers = len(validatedFilePaths)
				}
				//TODO: batch work here
			} else {
				for _, filePath := range validatedFilePaths {
					respURL, guid, err := getPresignedURL(function, filePath, uploadPath, includeSubDirName)
					if err != nil {
						log.Println(err.Error())
						continue
					}
					file, err := os.Open(filePath)
					if err != nil {
						log.Println("File open error")
						continue
					}
					req, bar, err := GenerateUploadRequest("", respURL, file)
					if err != nil {
						file.Close()
						log.Println("Error occurred during request generation: %s", err.Error())
						continue
					}
					uploadFile(req, bar, guid, filePath)
					historyFileMap[filePath] = guid
					file.Close()
				}
			}

			reqs := make([]*http.Request, 0)
			bars := make([]*pb.ProgressBar, 0)

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
	uploadNewCmd.Flags().BoolVar(&includeSubDirName, "include-subdirname", false, "Include subdirectory names in file name")
	RootCmd.AddCommand(uploadNewCmd)
}
