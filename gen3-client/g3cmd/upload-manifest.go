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
	"sync"

	"gopkg.in/cheggaaa/pb.v1"

	"github.com/spf13/cobra"
)

// doBatchHTTPClient executes a batch of HTTP PUT requests using worker pool. The default number of workers is 3
func doBatchHTTPClient(client *http.Client, workers int, requests ...*http.Request) (<-chan *http.Response, <-chan error) {
	if workers < 1 || workers > len(requests) {
		workers = len(requests)
	}

	// channels for requests, responses and errors
	reqch := make(chan *http.Request, len(requests))
	respch := make(chan *http.Response, len(requests))
	errch := make(chan error, len(requests))

	wg := sync.WaitGroup{}
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			for req := range reqch {
				resp, err := client.Do(req)
				if err != nil {
					errch <- err
				} else {
					respch <- resp
				}
			}
			wg.Done()
		}()
	}

	for _, req := range requests {
		reqch <- req
	}
	close(reqch)

	wg.Wait()
	close(respch)
	return respch, errch
}

func batchUpload(numParallel int, reqs []*http.Request, bars []*pb.ProgressBar) {
	pool, err := pb.StartPool(bars...)
	if err != nil {
		panic(err)
	}

	client := &http.Client{}
	_, errch := doBatchHTTPClient(client, numParallel, reqs...)

	wg := new(sync.WaitGroup)
	for _, bar := range bars {
		wg.Add(1)
		bar.Start()
		go func(cb *pb.ProgressBar) {
			for cb.Get() < cb.Total {
			}
			wg.Done()
		}(bar)
	}
	wg.Wait()

	if len(errch) > 0 {
		for err := range errch {
			if err != nil {
				fmt.Printf("Error: %s\n", err.Error())
			}
		}
	}

	pool.Stop()
	fmt.Printf("%d files uploaded.\n", len(reqs))
}

func getFullFilePath(filePath string, filename string) (string, error) {
	fi, err := os.Stat(filePath)
	if err != nil {
		fmt.Println(err)
		return "", err
	}
	switch mode := fi.Mode(); {
	case mode.IsDir():
		if strings.HasSuffix(filePath, "/") {
			return filePath + filename, nil
		} else {
			return filePath + "/" + filename, nil
		}
	case mode.IsRegular():
		return "", errors.New("in manifest upload mode filePath must be a dir")
	default:
		return "", errors.New("full file path creation unsuccessful")
	}
}

func init() {
	var manifestPath string
	var uploadPath string
	var fileType string
	var batch bool
	var numParallel int

	var uploadManifestCmd = &cobra.Command{
		Use:     "upload-manifest",
		Short:   "upload files from a specified manifest",
		Long:    `Gets a presigned URL for a file from a GUID and then uploads the specified file.`,
		Example: `./gen3-client upload-manifest --profile user1 --manifest manifest.tsv --upload-path=files/`,
		Run: func(cmd *cobra.Command, args []string) {
			var objects []ManifestObject

			manifestFile, err := os.Open(manifestPath)
			if err != nil {
				log.Fatalf("Failed to open manifest file\n")
				return
			}
			defer manifestFile.Close()

			switch {
			case strings.EqualFold(filepath.Ext(manifestPath), ".json"):
				manifestBytes, err := ioutil.ReadFile(manifestPath)
				if err != nil {
					log.Fatalf("Failed reading manifest %s, %v\n", manifestPath, err)
				}
				json.Unmarshal(manifestBytes, &objects)
			default:
				log.Fatalf("Unsupported manifast format")
				return
			}

			reqs := make([]*http.Request, 0)
			bars := make([]*pb.ProgressBar, 0)
			for _, object := range objects {
				guid := object.ObjectID
				// Here we are assuming the local filename will be the same as GUID
				filePath, err := getFullFilePath(uploadPath, object.ObjectID)
				if err != nil {
					log.Fatalf(err.Error())
					continue
				}

				if _, err := os.Stat(filePath); os.IsNotExist(err) {
					log.Fatalf("The file you specified \"%s\" does not exist locally.", filePath)
				}

				file, err := os.Open(filePath)
				if err != nil {
					log.Fatal("File Error")
				}
				defer file.Close()

				req, bar, err := GenerateUploadRequest(guid, file, fileType)
				if err != nil {
					log.Fatalf("Error occured during request generation: %s", err.Error())
					continue
				}
				if batch {
					reqs = append(reqs, req)
					bars = append(bars, bar)
				} else {
					uploadFile(req, bar, guid, filePath)
				}
			}
			if batch {
				batchUpload(numParallel, reqs, bars)
			}
		},
	}

	uploadManifestCmd.Flags().StringVar(&manifestPath, "manifest", "", "The manifest file to read from")
	uploadManifestCmd.MarkFlagRequired("manifest")
	uploadManifestCmd.Flags().StringVar(&uploadPath, "upload-path", "", "The directory in which contains files to be uploaded")
	uploadManifestCmd.MarkFlagRequired("upload-path")
	uploadManifestCmd.Flags().StringVar(&fileType, "file-type", "json", "Specify file type you're uploading with --file-type={json|tsv} (defaults to json)")
	uploadManifestCmd.Flags().BoolVar(&batch, "batch", true, "Upload in parallel")
	uploadManifestCmd.Flags().IntVar(&numParallel, "numparallel", 2, "Number of uploads to run in parallel")
	RootCmd.AddCommand(uploadManifestCmd)
}
