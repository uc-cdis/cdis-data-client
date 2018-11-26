package g3cmd

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"gopkg.in/cheggaaa/pb.v1"

	"github.com/spf13/cobra"

	"github.com/uc-cdis/gen3-client/gen3-client/jwt"
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
	// doBatchHTTPClient(client, numParallel, reqs...)

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

	// t := time.NewTicker(200 * time.Millisecond)

	// completed := 0
	// responses := make([]*http.Response, 0)
	// errors := make([]error, 0)
	// for completed < len(reqs) {
	// 	select {
	// 	case resp := <-respch:
	// 		if resp != nil {
	// 			responses = append(responses, resp)
	// 		}
	// 	case err := <-errch:
	// 		if err != nil {
	// 			errors = append(errors, err)
	// 		}

	// 	case <-t.C:
	// 		for i, resp := range responses {
	// 			if resp != nil {
	// 				responses[i] = nil
	// 				completed++
	// 			}
	// 		}

	// 		for i, err := range errors {
	// 			if err != nil {
	// 				fmt.Printf("Error: %s\n", err.Error())
	// 				errors[i] = nil
	// 				completed++
	// 			}
	// 		}
	// 		fmt.Println(completed)
	// 	}
	// }

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

func init() {
	var manifestPath string
	var uploadPath string
	var fileType string
	var batch bool
	var numParallel int

	var uploadManifestCmd = &cobra.Command{
		Use:   "upload-manifest",
		Short: "upload files from a specified manifest",
		Long: `Gets a presigned URL for a file from a GUID and then uploads the specified file.
	Examples: ./gen3-client upload-manifest --profile user1 --manifest manifest.tsv --upload-path=files/ 
	`,
		Run: func(cmd *cobra.Command, args []string) {

			request := new(jwt.Request)
			configure := new(jwt.Configure)
			function := new(jwt.Functions)

			function.Config = configure
			function.Request = request

			var objects []ManifestObject

			manifestFile, err := os.Open(manifestPath)
			if err != nil {
				panic(err)
			}
			defer manifestFile.Close()

			switch {
			case strings.EqualFold(filepath.Ext(manifestPath), ".tsv"):
				r := csv.NewReader(manifestFile)
				r.Comma = '\t'
				for {
					record, err := r.Read()
					if err == io.EOF {
						break
					} else if err != nil {
						log.Fatalf("TSV parse error\n")
					}
					objects = append(objects, ManifestObject{ObjectID: record[0], SubjectID: record[1]})
				}
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

			if batch {
				reqs := make([]*http.Request, 0)
				bars := make([]*pb.ProgressBar, 0)
				for _, object := range objects {
					endPointPostfix := "/user/data/upload/" + object.ObjectID
					respURL, _, err := function.DoRequestWithSignedHeader(profile, "", endPointPostfix, nil)

					file, err := os.Open(uploadPath + "/" + object.SubjectID)
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
				}
				batchUpload(numParallel, reqs, bars)
			} else {
				for _, object := range objects {
					endPointPostfix := "/user/data/upload/" + object.ObjectID
					respURL, _, err := function.DoRequestWithSignedHeader(profile, "", endPointPostfix, nil)

					if err != nil {
						if strings.Contains(err.Error(), "The provided guid") {
							log.Printf("Upload error: %s\n", err)
						} else {
							log.Fatalf("Fatal upload error: %s\n", err)
						}
					} else {
						filePath := uploadPath + "/" + object.SubjectID
						uploadFile(object.ObjectID, filePath, fileType, respURL)
					}
				}
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
