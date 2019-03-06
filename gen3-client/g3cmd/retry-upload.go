package g3cmd

import (
	"fmt"
	"log"
	"math"
	"os"
	"path"
	"time"

	"github.com/spf13/cobra"
	"github.com/uc-cdis/gen3-client/gen3-client/commonUtils"
	"github.com/uc-cdis/gen3-client/gen3-client/logs"
)

// MaxRetryCount is the maximum retry number per record
const MaxRetryCount = 5
const maxWaitTime = 300

func getWaitTime(retryCount int) time.Duration {
	exponentialWaitTime := math.Pow(2, float64(retryCount))
	return time.Duration(math.Min(exponentialWaitTime, float64(maxWaitTime))) * time.Second
}

func retryUpload(failedLogMap map[string]commonUtils.RetryObject, includeSubDirName bool, uploadPath string) {
	var guid string
	var filename string
	var err error
	fmt.Println()
	if len(failedLogMap) == 0 {
		log.Println("No failed file in log, no need to retry upload.")
		return
	}

	log.Println("Retry upload has started...")
	retryObjCh := make(chan commonUtils.RetryObject, len(failedLogMap))
	for _, v := range failedLogMap {
		if logs.ExistsInSucceededLog(v.FilePath) {
			log.Println("File \"" + v.FilePath + "\" has been found in local submission history and has be skipped for preventing duplicated submissions.")
			continue
		}
		retryObjCh <- v
	}
	log.Printf("%d records has been sent to the retry channel\n\n", len(retryObjCh))
	if len(retryObjCh) == 0 {
		return
	}

	for ro := range retryObjCh {
		log.Printf("#%d retry of record %s\n", ro.RetryCount, ro.FilePath)
		if ro.PresignedURL == "" {
			ro.PresignedURL, guid, filename, err = GeneratePresignedURL(uploadPath, ro.FilePath, includeSubDirName)
			if err != nil {
				ro.RetryCount++
				logs.AddToFailedLogMap(ro.FilePath, guid, ro.PresignedURL, ro.RetryCount, false)
				log.Println(err.Error())
				if ro.RetryCount < MaxRetryCount { // try another time
					retryObjCh <- ro
				} else {
					logs.IncrementScore(logs.ScoreBoardLen - 1) // inevitable failure
					if (len(retryObjCh)) == 0 {
						close(retryObjCh)
					}
				}
				continue
			}
		} else {
			filename = path.Base(ro.FilePath)
		}

		furObject := commonUtils.FileUploadRequestObject{FilePath: ro.FilePath, Filename: filename, GUID: guid, PresignedURL: ro.PresignedURL}
		file, err := os.Open(ro.FilePath)
		if err != nil {
			ro.RetryCount++
			logs.AddToFailedLogMap(furObject.FilePath, furObject.GUID, furObject.PresignedURL, ro.RetryCount, false)
			log.Println("File open error: " + err.Error())
			if ro.RetryCount < MaxRetryCount {
				retryObjCh <- ro
			} else {
				logs.IncrementScore(logs.ScoreBoardLen - 1)
				if (len(retryObjCh)) == 0 {
					close(retryObjCh)
				}
			}
			continue
		}

		furObject, err = GenerateUploadRequest(furObject, file)
		if err != nil {
			ro.RetryCount++
			file.Close()
			logs.AddToFailedLogMap(furObject.FilePath, furObject.GUID, furObject.PresignedURL, ro.RetryCount, false)
			log.Println("Error occurred during request generation: " + err.Error())
			if ro.RetryCount < MaxRetryCount {
				retryObjCh <- ro
			} else {
				logs.IncrementScore(logs.ScoreBoardLen - 1)
				if (len(retryObjCh)) == 0 {
					close(retryObjCh)
				}
			}
			continue
		}

		log.Printf("Sleep for %.0f seconds\n", getWaitTime(ro.RetryCount).Seconds())
		time.Sleep(getWaitTime(ro.RetryCount)) // exponential wait for retry
		err = uploadFile(furObject, ro.RetryCount)
		if err != nil {
			ro.RetryCount++
			file.Close()
			logs.AddToFailedLogMap(furObject.FilePath, furObject.GUID, furObject.PresignedURL, ro.RetryCount, false)
			log.Println(err.Error())
			if ro.RetryCount < MaxRetryCount {
				retryObjCh <- ro
			} else {
				logs.IncrementScore(logs.ScoreBoardLen - 1)
				if (len(retryObjCh)) == 0 {
					close(retryObjCh)
				}
			}
			continue
		}
		logs.IncrementScore(ro.RetryCount + 1)
		if (len(retryObjCh)) == 0 {
			close(retryObjCh)
			log.Println("Retry channel has been closed")
		}
	}
}

func init() {
	var failedLogPath string
	var includeSubDirName bool
	var uploadPath string
	var retryUploadCmd = &cobra.Command{
		Use:     "retry-upload",
		Short:   "Retry upload file(s) to object storage.",
		Long:    `Re-submit files found in a given failed log by using sequential (non-batching) uploading and exponential backoff.`,
		Example: "For retrying file upload:\n./gen3-client retry-upload --profile=<profile-name> --failed-log-path=<path-to-failed-log>\n",
		Run: func(cmd *cobra.Command, args []string) {
			if includeSubDirName && uploadPath == "" {
				fmt.Println("Error: in retry upload mode, you need to specify --uploadPath option to the parent directory of data files if you set --includeSubDirName=true")
				return
			}
			failedLogPath = commonUtils.ParseRootPath(failedLogPath)
			logs.LoadFailedLogFile(failedLogPath)
			logs.InitScoreBoard(MaxRetryCount)
			retryUpload(logs.GetFailedLogMap(), includeSubDirName, uploadPath)
			logs.CloseAll()
			logs.PrintScoreBoard()
		},
	}

	retryUploadCmd.Flags().StringVar(&failedLogPath, "failed-log-path", "", "The path to the failed log file.")
	retryUploadCmd.MarkFlagRequired("failed-log-path")
	retryUploadCmd.Flags().StringVar(&uploadPath, "upload-path", "", "The directory or file in which contains file(s) to be uploaded")
	retryUploadCmd.Flags().BoolVar(&includeSubDirName, "include-subdirname", false, "Include subdirectory names in file name")
	RootCmd.AddCommand(retryUploadCmd)
}
