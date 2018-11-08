package g3cmd

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/uc-cdis/gen3-client/gen3-client/jwt"
)

func init() {
	var uploadPath string
	var fileType string
	var batch bool
	var numParallel int
	var files []string

	var uploadNewCmd = &cobra.Command{
		Use:   "upload-new",
		Short: "upload file(s) with a new flow.",
		Long: `Gets a presigned URL for a file and then uploads the specified file.
	Examples: ./gen3-client upload-new --profile user1 --upload-path=files/ 
	`,
		Run: func(cmd *cobra.Command, args []string) {

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
					files = append(files, filepath.Join(uploadPath, file.Name()))
				}
			} else {
				files = append(files, uploadPath)
			}

			if batch {
				reqs := make([]*http.Request, 0)
				for _, file := range files {
					endPointPostfix := "/user/data/upload/" + filepath.Base(file)
					respURL, UUID, err := function.DoRequestWithSignedHeader(profile, "", endPointPostfix)

					data, err := ioutil.ReadFile(file)
					if err != nil {
						fmt.Println(err.Error())
						break
					}
					body := bytes.NewBufferString(string(data[:]))
					contentType := "application/json"
					if fileType == "tsv" {
						contentType = "text/tab-separated-values"
					}
					req, _ := http.NewRequest(http.MethodPut, respURL, body)
					req.Header.Set("content_type", contentType)
					reqs = append(reqs, req)
					//TODO: save uuid
					fmt.Println(UUID)
				}
				batchUpload(numParallel, reqs, nil)
			} else {
				for _, file := range files {
					endPointPostfix := "/user/data/upload/" + filepath.Base(file)
					respURL, UUID, err := function.DoRequestWithSignedHeader(profile, "", endPointPostfix)

					if err != nil {
						log.Fatalf("Fatal upload error: %s\n", err)
					} else {
						uploadFile("", uploadPath+"/"+UUID, fileType, respURL)
					}
				}
			}
		},
	}

	uploadNewCmd.Flags().StringVar(&uploadPath, "upload-path", "", "The directory or file in which contains file(s) to be uploaded")
	uploadNewCmd.MarkFlagRequired("upload-path")
	uploadNewCmd.Flags().StringVar(&fileType, "file-type", "json", "Specify file type you're uploading with --file-type={json|tsv} (defaults to json)")
	uploadNewCmd.Flags().BoolVar(&batch, "batch", true, "Upload in parallel")
	uploadNewCmd.Flags().IntVar(&numParallel, "numparallel", 3, "Number of uploads to run in parallel")
	RootCmd.AddCommand(uploadNewCmd)
}
