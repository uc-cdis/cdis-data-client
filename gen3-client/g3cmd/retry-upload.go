package g3cmd

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"
	"github.com/uc-cdis/gen3-client/gen3-client/commonUtils"
	"github.com/uc-cdis/gen3-client/gen3-client/logs"
)

func updateRetryObject(ro *commonUtils.RetryObject, filePath string, filename string, fileMetadata commonUtils.FileMetadata, guid string, retryCount int, isMultipart bool) {
	ro.FilePath = filePath
	ro.Filename = filename
	ro.FileMetadata = fileMetadata
	ro.GUID = guid
	ro.RetryCount = retryCount
	ro.Multipart = isMultipart
}

func handleFailedRetry(ro commonUtils.RetryObject, retryObjCh chan commonUtils.RetryObject, err error, isMuted bool) {
	gen3Interface := NewGen3Interface()
	logs.AddToFailedLog(ro.FilePath, ro.Filename, ro.FileMetadata, ro.GUID, ro.RetryCount, ro.Multipart, isMuted)
	if err != nil {
		log.Println(err.Error())
	}
	if ro.RetryCount < MaxRetryCount { // try another time
		retryObjCh <- ro
	} else {
		if ro.GUID != "" {
			msg, err := DeleteRecord(gen3Interface, ro.GUID)
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

func retryUpload(failedLogMap map[string]commonUtils.RetryObject) {

	gen3Interface := NewGen3Interface()

	var guid string
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
			log.Println("File \"" + v.FilePath + "\" has been found in local submission history and has been skipped to prevent duplicated submissions.")
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
			msg, err := DeleteRecord(gen3Interface, ro.GUID)
			if err == nil {
				log.Println(msg)
			} else {
				log.Println(err.Error())
			}
		}

		if ro.Filename == "" {
			filePath, _ := commonUtils.GetAbsolutePath(ro.FilePath)
			filename := filepath.Base(filePath)
			updateRetryObject(&ro, filePath, filename, ro.FileMetadata, ro.GUID, ro.RetryCount, true)
		}

		if ro.Multipart {
			fileInfo := FileInfo{FilePath: ro.FilePath, Filename: ro.Filename}
			err = multipartUpload(gen3Interface, fileInfo, ro.RetryCount, ro.Bucket)
			if err != nil {
				updateRetryObject(&ro, ro.FilePath, ro.Filename, ro.FileMetadata, ro.GUID, ro.RetryCount, true)
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
			presignedURL, guid, err = GeneratePresignedURL(gen3Interface, ro.Filename, ro.FileMetadata, ro.Bucket)
			if err != nil {
				updateRetryObject(&ro, ro.FilePath, ro.Filename, ro.FileMetadata, guid, ro.RetryCount, false)
				handleFailedRetry(ro, retryObjCh, err, true)
				continue
			}
			furObject := commonUtils.FileUploadRequestObject{FilePath: ro.FilePath, Filename: ro.Filename, FileMetadata: ro.FileMetadata, GUID: guid, PresignedURL: presignedURL}
			file, err := os.Open(ro.FilePath)
			if err != nil {
				updateRetryObject(&ro, furObject.FilePath, furObject.Filename, furObject.FileMetadata, ro.GUID, ro.RetryCount, false)
				handleFailedRetry(ro, retryObjCh, err, false)
				continue
			}
			fi, err := file.Stat()
			if err != nil {
				updateRetryObject(&ro, furObject.FilePath, furObject.Filename, furObject.FileMetadata, ro.GUID, ro.RetryCount, false)
				handleFailedRetry(ro, retryObjCh, err, false)
				file.Close()
				continue
			}
			if fi.Size() > FileSizeLimit { // guard for files, always check file size during retry upload
				updateRetryObject(&ro, furObject.FilePath, furObject.Filename, furObject.FileMetadata, guid, ro.RetryCount, true)
				err = fmt.Errorf("File size for %s is greater than the single part upload limit, will retry using multipart upload", furObject.Filename)
				handleFailedRetry(ro, retryObjCh, err, false)
				file.Close()
				continue
			}

			furObject, err = GenerateUploadRequest(gen3Interface, furObject, file)
			if err != nil {
				updateRetryObject(&ro, furObject.FilePath, furObject.Filename, furObject.FileMetadata, furObject.GUID, ro.RetryCount, false)
				handleFailedRetry(ro, retryObjCh, err, false)
				file.Close()
				continue
			}

			err = uploadFile(furObject, ro.RetryCount)
			if err != nil {
				updateRetryObject(&ro, furObject.FilePath, furObject.Filename, furObject.FileMetadata, furObject.GUID, ro.RetryCount, false)
				handleFailedRetry(ro, retryObjCh, err, false)
				file.Close()
				continue
			}
			logs.DeleteFromFailedLog(furObject.FilePath, true)
			logs.IncrementScore(ro.RetryCount)
			file.Close()
			if (len(retryObjCh)) == 0 {
				close(retryObjCh)
				log.Println("Retry channel has been closed")
			}
		}
	}
}

func init() {
	var failedLogPath string
	var retryUploadCmd = &cobra.Command{
		Use:     "retry-upload",
		Short:   "Retry upload file(s) to object storage.",
		Long:    `Re-submit files found in a given failed log by using sequential (non-batching) uploading and exponential backoff.`,
		Example: "For retrying file upload:\n./gen3-client retry-upload --profile=<profile-name> --failed-log-path=<path-to-failed-log>\n",
		Run: func(cmd *cobra.Command, args []string) {
			// initialize transmission logs
			logs.InitSucceededLog(profile)
			logs.InitFailedLog(profile)
			logs.SetToBoth()
			logs.InitScoreBoard(MaxRetryCount)
			profileConfig = conf.ParseConfig(profile)

			failedLogPath = commonUtils.ParseRootPath(failedLogPath)
			logs.LoadFailedLogFile(failedLogPath)
			retryUpload(logs.GetFailedLogMap())
			logs.PrintScoreBoard()
			logs.CloseAll()
		},
	}

	retryUploadCmd.Flags().StringVar(&profile, "profile", "", "Specify profile to use")
	retryUploadCmd.MarkFlagRequired("profile") //nolint:errcheck
	retryUploadCmd.Flags().StringVar(&failedLogPath, "failed-log-path", "", "The path to the failed log file.")
	retryUploadCmd.MarkFlagRequired("failed-log-path") //nolint:errcheck
	RootCmd.AddCommand(retryUploadCmd)
}
