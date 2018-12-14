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
	"github.com/uc-cdis/gen3-client/gen3-client/jwt"
	pb "gopkg.in/cheggaaa/pb.v1"
)

var historyFile string
var historyFileMap map[string]string

func init() {
	var uploadPath string
	var batch bool
	var numParallel int

	var uploadNewCmd = &cobra.Command{
		Use:     "upload",
		Short:   "upload file(s) to object storage.",
		Long:    `Gets a presigned URL for each file and then uploads the specified file(s).`,
		Example: "./gen3-client upload --profile=<profile-name> --upload-path=<path-to-files/data.bam>\nor\n./gen3-client upload --profile=<profile-name> --upload-path=<path-to-files/*.bam>\nor\n./gen3-client upload --profile=<profile-name> --upload-path=<path-to-files/*/folder/*>",
		Run: func(cmd *cobra.Command, args []string) {
			initHistory()

			request := new(jwt.Request)
			configure := new(jwt.Configure)
			function := new(jwt.Functions)

			function.Config = configure
			function.Request = request

			filePaths, err := filepath.Glob(uploadPath) // Generating all possible file paths
			for _, filePath := range filePaths {
				file, err := os.Open(filePath)
				if err != nil {
					log.Fatal("File Error")
				}

				if fi, _ := file.Stat(); fi.IsDir() {
					err = filepath.Walk(filePath, func(path string, fileInfo os.FileInfo, err error) error {
						if err != nil {
							return err
						}
						if !fileInfo.IsDir() {
							filePaths = append(filePaths, path)
						}
						return nil
					})
				}
				file.Close()
			}
			if err != nil {
				log.Fatalf(err.Error())
			}
			if len(filePaths) == 0 {
				log.Fatalf("Error when parsing file paths.")
			}

			reqs := make([]*http.Request, 0)
			bars := make([]*pb.ProgressBar, 0)
			for _, filePath := range filePaths {
				file, err := os.Open(filePath)
				if err != nil {
					log.Fatal("File Error")
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

				respURL, guid, err := function.DoRequestWithSignedHeader(profile, "", endPointPostfix, objectBytes)

				if respURL == "" || guid == "" {
					log.Fatalf("Error has occured during presigned URL or GUID generation.")
				}

				if _, err := os.Stat(filePath); os.IsNotExist(err) {
					log.Fatalf("The file you specified \"%s\" does not exist locally.", filePath)
				}

				req, bar, err := GenerateUploadRequest("", respURL, file)
				if err != nil {
					log.Fatalf("Error occured during request generation: %s", err.Error())
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

func initHistory() {
	home, err := homedir.Dir()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	historyFile = home + "/.gen3/" + profile + "_history.json"

	file, _ := os.OpenFile(historyFile, os.O_RDWR|os.O_CREATE, 0666)
	fi, err := file.Stat()
	if err != nil {
		panic(err)
	}

	historyFileMap = make(map[string]string)
	if fi.Size() > 0 {
		data, err := ioutil.ReadAll(file)
		if err != nil {
			log.Fatal(err)
		}

		err = json.Unmarshal(data, &historyFileMap)
		if err != nil {
			panic(err)
		}
	}
}
