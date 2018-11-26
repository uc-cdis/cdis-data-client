package g3cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"

	homedir "github.com/mitchellh/go-homedir"
	"github.com/spf13/cobra"
	pb "gopkg.in/cheggaaa/pb.v1"

	"github.com/uc-cdis/gen3-client/gen3-client/jwt"
)

var historyFile string
var historyFileMap map[string]interface{}

func init() {
	var uploadPath string
	var fileType string
	var batch bool
	var numParallel int
	var filePaths []string

	var uploadNewCmd = &cobra.Command{
		Use:   "upload-new",
		Short: "upload file(s) with a new flow.",
		Long: `Gets a presigned URL for a file and then uploads the specified file.
	Examples: ./gen3-client upload-new --profile user1 --upload-path=files/ 
	`,
		Run: func(cmd *cobra.Command, args []string) {
			initHistory()

			request := new(jwt.Request)
			configure := new(jwt.Configure)
			function := new(jwt.Functions)

			function.Config = configure
			function.Request = request

			fi, err := os.Stat(uploadPath)
			if err != nil {
				panic(err)
			}
			if fi.IsDir() {
				dirFiles, err := ioutil.ReadDir(uploadPath)
				if err != nil {
					log.Fatal(err)
				}
				for _, file := range dirFiles {
					filePaths = append(filePaths, filepath.Join(uploadPath, file.Name()))
				}
			} else {
				filePaths = append(filePaths, uploadPath)
			}

			reqs := make([]*http.Request, 0)
			bars := make([]*pb.ProgressBar, 0)
			for _, filePath := range filePaths {
				_, present := historyFileMap[filePath]
				if present {
					fmt.Printf("File %s has been found in local submission history and has be skipped for preventing duplicated submissions.\n", filePath)
					continue
				}
				endPointPostfix := "/user/data/upload"
				dataBody := bytes.NewBufferString("{\"filename\": \"" + filepath.Base(filePath) + "\"}")
				respURL, GUID, err := function.DoRequestWithSignedHeader(profile, "", endPointPostfix, dataBody)

				if batch {
					file, err := os.Open(filePath)
					if err != nil {
						log.Fatal("File Error")
					}
					defer file.Close()

					req, bar, err := GenerateUploadRequest(file, fileType, respURL)

					if err != nil {
						fmt.Println(err.Error())
						break
					}

					reqs = append(reqs, req)
					bars = append(bars, bar)
					fmt.Println(GUID)
				} else {
					if err != nil {
						log.Fatalf("Fatal upload error: %s\n", err)
					} else {
						uploadFile(GUID, filePath, fileType, respURL)
					}
				}
				historyFileMap[filePath] = GUID
			}

			if batch {
				batchUpload(numParallel, reqs, nil)
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
	uploadNewCmd.Flags().StringVar(&fileType, "file-type", "json", "Specify file type you're uploading with --file-type={json|tsv} (defaults to json)")
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
	fmt.Println(historyFile)

	file, _ := os.OpenFile(historyFile, os.O_RDWR|os.O_CREATE, 0666)
	fi, err := file.Stat()
	if err != nil {
		panic(err)
	}

	if fi.Size() > 0 {
		historyFileMap = make(map[string]interface{})

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
