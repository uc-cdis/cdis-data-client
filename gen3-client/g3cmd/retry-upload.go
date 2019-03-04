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
	presignedURL string
	retryCount   int
}

const MaxRetryCount = 5
const maxWaitTime = 300000

var failedLogPath string
var err error
var includeSubDirName bool

func getWaitTime(retryCount int) time.Duration {
	exponentialWaitTime := math.Pow(2, float64(retryCount)) * 1000
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
		retryObjCh <- retryObject{filePath: f, presignedURL: u, retryCount: 0}
	}

	for ro := range retryObjCh {
		if ro.presignedURL == "" {
			ro.presignedURL, guid, err = GeneratePresignedURL(ro.filePath, includeSubDirName)
			if err != nil {
				logs.AddToFailedLogMap(ro.filePath, ro.presignedURL, false)
				log.Println(err.Error())
				ro.retryCount++
				if ro.retryCount < MaxRetryCount { // try another time
					retryObjCh <- ro
				} else {
					logs.IncrementScore(len(logs.ScoreBoard) - 1) // inevitable failure
				}
				continue
			}
		}
		furObject := FileUploadRequestObject{FilePath: ro.filePath, GUID: guid, PresignedURL: ro.presignedURL}
		file, err := os.Open(ro.filePath)
		if err != nil {
			logs.AddToFailedLogMap(furObject.FilePath, furObject.PresignedURL, false)
			log.Println("File open error: " + err.Error())
			ro.retryCount++
			if ro.retryCount < MaxRetryCount {
				retryObjCh <- ro
			} else {
				logs.IncrementScore(len(logs.ScoreBoard) - 1)
			}
			continue
		}
		furObject, err = GenerateUploadRequest(furObject, file)
		if err != nil {
			file.Close()
			logs.AddToFailedLogMap(furObject.FilePath, furObject.PresignedURL, false)
			log.Println("Error occurred during request generation: " + err.Error())
			ro.retryCount++
			if ro.retryCount < MaxRetryCount {
				retryObjCh <- ro
			} else {
				logs.IncrementScore(len(logs.ScoreBoard) - 1)
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
			if ro.retryCount < MaxRetryCount {
				retryObjCh <- ro
			} else {
				logs.IncrementScore(len(logs.ScoreBoard) - 1)
			}
			continue
		}
		logs.IncrementScore(ro.retryCount + 1)
	}
}

func init() {
	var retryUploadCmd = &cobra.Command{
		Use:     "retry-upload",
		Short:   "retry upload file(s) to object storage.",
		Example: "For retrying file upload:\n./gen3-client retry-upload --profile=<profile-name> --failed-log-path=<path-to-failed-log>\n",
		Run: func(cmd *cobra.Command, args []string) {
			logs.LoadFailedLogFile(failedLogPath)
			retryUpload(logs.GetFailedLogMap())
		},
	}

	retryUploadCmd.Flags().StringVar(&failedLogPath, "failed-log-path", "", "The path to the failed log file.")
	retryUploadCmd.MarkFlagRequired("failed-log-path")
	retryUploadCmd.Flags().BoolVar(&includeSubDirName, "include-subdirname", false, "Include subdirectory names in file name")
	RootCmd.AddCommand(retryUploadCmd)
}
