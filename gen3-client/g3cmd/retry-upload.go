package g3cmd

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/spf13/cobra"
	"github.com/uc-cdis/gen3-client/gen3-client/commonUtils"
	"github.com/uc-cdis/gen3-client/gen3-client/logs"
)

func updateRetryObject(ro commonUtils.RetryObject, filePath string, guid string, retryCount int, isMultipart bool) {
	ro.FilePath = filePath
	ro.GUID = guid
	ro.RetryCount = retryCount
	ro.Multipart = isMultipart
}

func handleFailedRetry(ro commonUtils.RetryObject, retryObjCh chan commonUtils.RetryObject, err error, isMuted bool) {
	ro.RetryCount++
	logs.AddToFailedLogMap(ro.FilePath, ro.GUID, ro.RetryCount, ro.Multipart, isMuted)
	if err != nil {
		log.Println(err.Error())
	}
	if ro.RetryCount < MaxRetryCount { // try another time
		retryObjCh <- ro
	} else {
		if ro.GUID != "" {
			msg, err := DeleteRecord(ro.GUID)
			if err == nil {
				log.Println(msg)
			} else {
				log.Println(err.Error())
			}
		}
		logs.IncrementScore(logs.ScoreBoardLen - 1) // inevitable failure
		if (len(retryObjCh)) == 0 {
			close(retryObjCh)
			log.Println("Retry channel has been closed")
		}
	}
}

func retryUpload(failedLogMap map[string]commonUtils.RetryObject, uploadPath string, includeSubDirName bool) {
	var guid string
	var filename string
	var presignedURL string
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
		ro.RetryCount++
		log.Printf("#%d retry of record %s\n", ro.RetryCount, ro.FilePath)
		log.Printf("Sleep for %.0f seconds\n", GetWaitTime(ro.RetryCount).Seconds())
		time.Sleep(GetWaitTime(ro.RetryCount)) // exponential wait for retry

		if ro.GUID != "" {
			msg, err := DeleteRecord(ro.GUID)
			if err == nil {
				log.Println(msg)
			} else {
				log.Println(err.Error())
			}
		}

		if ro.Multipart {
			err = multipartUpload(uploadPath, ro.FilePath, includeSubDirName, ro.RetryCount)
			if err != nil {
				updateRetryObject(ro, ro.FilePath, ro.GUID, ro.RetryCount, true)
				handleFailedRetry(ro, retryObjCh, err, true)
				continue
			} else { // succeeded
				logs.IncrementScore(ro.RetryCount)
				if (len(retryObjCh)) == 0 {
					close(retryObjCh)
					log.Println("Retry channel has been closed")
				}
			}
		} else {
			presignedURL, guid, filename, err = GeneratePresignedURL(uploadPath, ro.FilePath, includeSubDirName)
			if err != nil {
				updateRetryObject(ro, ro.FilePath, guid, ro.RetryCount, false)
				handleFailedRetry(ro, retryObjCh, err, true)
				continue
			}
			furObject := commonUtils.FileUploadRequestObject{FilePath: ro.FilePath, Filename: filename, GUID: guid, PresignedURL: presignedURL}
			file, err := os.Open(ro.FilePath)
			fi, err := file.Stat()
			if err != nil {
				updateRetryObject(ro, furObject.FilePath, furObject.GUID, ro.RetryCount, false)
				handleFailedRetry(ro, retryObjCh, err, false)
				file.Close()
				continue
			}
			if fi.Size() > FileSizeLimit { // guard for files, always check file size during retry upload
				updateRetryObject(ro, furObject.FilePath, guid, ro.RetryCount, true)
				err = fmt.Errorf("File size for %s is greater than the single part upload limit, will retry using multipart upload", furObject.Filename)
				handleFailedRetry(ro, retryObjCh, err, false)
				file.Close()
				continue
			}

			furObject, err = GenerateUploadRequest(furObject, file)
			if err != nil {
				updateRetryObject(ro, furObject.FilePath, furObject.GUID, ro.RetryCount, false)
				handleFailedRetry(ro, retryObjCh, err, false)
				file.Close()
				continue
			}

			err = uploadFile(furObject, ro.RetryCount)
			if err != nil {
				updateRetryObject(ro, furObject.FilePath, furObject.GUID, ro.RetryCount, false)
				handleFailedRetry(ro, retryObjCh, err, false)
				file.Close()
				continue
			}
			logs.DeleteFromFailedLogMap(furObject.FilePath, true)
			logs.IncrementScore(ro.RetryCount)
			file.Close()
			if (len(retryObjCh)) == 0 {
				close(retryObjCh)
				log.Println("Retry channel has been closed")
			}
		}
	}
	logs.WriteToFailedLog()
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
			retryUpload(logs.GetFailedLogMap(), uploadPath, includeSubDirName)
			logs.CloseAll()
			logs.PrintScoreBoard()
		},
	}

	retryUploadCmd.Flags().StringVar(&profile, "profile", "", "Specify profile to use")
	retryUploadCmd.MarkFlagRequired("profile")
	retryUploadCmd.Flags().StringVar(&failedLogPath, "failed-log-path", "", "The path to the failed log file.")
	retryUploadCmd.MarkFlagRequired("failed-log-path")
	retryUploadCmd.Flags().StringVar(&uploadPath, "upload-path", "", "The directory or file in which contains file(s) to be uploaded")
	retryUploadCmd.Flags().BoolVar(&includeSubDirName, "include-subdirname", false, "Include subdirectory names in file name")
	RootCmd.AddCommand(retryUploadCmd)
}
