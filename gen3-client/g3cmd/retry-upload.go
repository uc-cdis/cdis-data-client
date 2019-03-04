package g3cmd

import (
	"fmt"
	"log"
	"math"
	"os"
	"time"

	"github.com/spf13/cobra"
	"github.com/uc-cdis/gen3-client/gen3-client/logs"
)

type retryObject struct {
	filePath     string
	presignedUrl string
	retryCount   int
}

const maxRetryCount = 5
const maxWaitTime = 300000

var failedLogPath string
var err error
var includeSubDirName bool

func getWaitTime(retryCount int) time.Duration {
	exponentialWaitTime := math.Pow(2, float64(retryCount))
	return time.Duration(math.Max(exponentialWaitTime, float64(maxWaitTime))) * time.Millisecond
}

func retryUpload(failedLogMap map[string]string) {
	var guid string
	if len(failedLogMap) == 0 {
		fmt.Println("No failed file in log, aborting...")
		return
	}
	retryObjCh := make(chan retryObject, len(failedLogMap))
	for f, u := range failedLogMap {
		retryObjCh <- retryObject{filePath: f, presignedUrl: u, retryCount: 0}
	}

	for ro := range retryObjCh {
		if ro.presignedUrl == "" {
			ro.presignedUrl, guid, err = GeneratePresignedURL(ro.filePath, includeSubDirName)
			if err != nil {
				logs.AddToFailedLogMap(ro.filePath, ro.presignedUrl, false)
				log.Println(err.Error())
				ro.retryCount++
				if ro.retryCount < maxRetryCount {
					retryObjCh <- ro
				}
				continue
			}
		}
		furObject := FileUploadRequestObject{FilePath: ro.filePath, GUID: guid, PresignedURL: ro.presignedUrl}
		file, err := os.Open(ro.filePath)
		if err != nil {
			logs.AddToFailedLogMap(furObject.FilePath, furObject.PresignedURL, false)
			log.Println("File open error: " + err.Error())
			ro.retryCount++
			if ro.retryCount < maxRetryCount {
				retryObjCh <- ro
			}
			continue
		}
		furObject, err = GenerateUploadRequest(furObject, file)
		if err != nil {
			file.Close()
			logs.AddToFailedLogMap(furObject.FilePath, furObject.PresignedURL, false)
			log.Println("Error occurred during request generation: " + err.Error())
			ro.retryCount++
			if ro.retryCount < maxRetryCount {
				retryObjCh <- ro
			}
			continue
		}

		time.Sleep(getWaitTime(ro.retryCount)) // exponential wait for retry
		err = uploadFile(furObject)
		if err != nil {
			file.Close()
			logs.AddToFailedLogMap(furObject.FilePath, furObject.PresignedURL, false)
			log.Println(err.Error())
			ro.retryCount++
			if ro.retryCount < maxRetryCount {
				retryObjCh <- ro
			}
			continue
		}
	}
}

func init() {
	var retryUploadCmd = &cobra.Command{
		Use:     "retry-upload",
		Short:   "retry upload file(s) to object storage.",
		Example: "For retrying file upload:\n./gen3-client retry-upload --profile=<profile-name> --failed-log-path=<path-to-failed-log>\n",
		Run: func(cmd *cobra.Command, args []string) {
			retryUpload(logs.GetFailedLogMap())
		},
	}

	retryUploadCmd.Flags().StringVar(&failedLogPath, "failed-log-path", "", "The path to the failed log file.")
	retryUploadCmd.MarkFlagRequired("failed-log-path")
	retryUploadCmd.Flags().BoolVar(&includeSubDirName, "include-subdirname", false, "Include subdirectory names in file name")
	RootCmd.AddCommand(retryUploadCmd)
}
