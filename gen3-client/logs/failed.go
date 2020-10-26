package logs

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"os"
	"sync"
	"time"

	"github.com/uc-cdis/gen3-client/gen3-client/commonUtils"
)

var failedLogFilename string
var failedLogFileMap map[string]commonUtils.RetryObject
var failedLogFile *os.File
var failedLogLock sync.Mutex
var err error

func InitFailedLog(profile string) {
	failedLogFilename = MainLogPath + profile + "_failed_log_" + time.Now().Format("20060102150405MST") + ".json"

	failedLogFile, err = os.OpenFile(failedLogFilename, os.O_RDWR|os.O_CREATE, 0766)
	if err != nil {
		failedLogFile.Close()
		log.Fatal("Error occurred when opening file \"" + failedLogFilename + "\": " + err.Error())
	}
	log.Println("Local failed log file \"" + failedLogFilename + "\" has opened")

	failedLogFileMap = make(map[string]commonUtils.RetryObject)
}

func LoadFailedLogFile(filePath string) {
	file, err := os.OpenFile(filePath, os.O_RDONLY, 0766)
	if err != nil {
		file.Close()
		failedLogFile.Close()
		log.Fatal("Error occurred when opening file \"" + file.Name() + "\": " + err.Error())
	}
	fi, err := file.Stat()
	if err != nil {
		file.Close()
		failedLogFile.Close()
		log.Fatal("Error occurred when opening file \"" + file.Name() + "\": " + err.Error())
	}
	log.Println("Failed log file \"" + file.Name() + "\" has been opened for read")

	if fi.Size() > 0 {
		var tempRetryObjectSlice []commonUtils.RetryObject
		data, err := ioutil.ReadAll(file)
		if err != nil {
			file.Close()
			failedLogFile.Close()
			log.Fatal("Error occurred when reading from file \"" + file.Name() + "\": " + err.Error())
		}

		err = json.Unmarshal(data, &tempRetryObjectSlice)
		if err != nil {
			file.Close()
			failedLogFile.Close()
			log.Fatal("Error occurred when unmarshaling from JSON objects: " + err.Error())
		}

		for _, ro := range tempRetryObjectSlice {
			failedLogFileMap[ro.FilePath] = ro
		}
	}
}

func IsFailedLogMapEmpty() bool {
	return len(failedLogFileMap) == 0
}

func GetFailedLogMap() map[string]commonUtils.RetryObject {
	return failedLogFileMap
}

func AddToFailedLog(filePath string, filename string, metadata commonUtils.FileMetadata, guid string, retryCount int, isMultipart bool, isMuted bool) {
	failedLogLock.Lock()
	defer failedLogLock.Unlock()
	failedLogFileMap[filePath] = commonUtils.RetryObject{FilePath: filePath, Filename: filename, FileMetadata: metadata, GUID: guid, RetryCount: retryCount, Multipart: isMultipart}
	if !isMuted {
		log.Printf("Failed file entry added for %s\n", filePath)
	}
	writeToFailedLog()
}

func DeleteFromFailedLog(filePath string, isMuted bool) {
	failedLogLock.Lock()
	defer failedLogLock.Unlock()
	delete(failedLogFileMap, filePath)
	if !isMuted {
		log.Printf("Failed file entry deleted for %s\n", filePath)
	}
	writeToFailedLog()
}

func writeToFailedLog() {
	var tempSlice []commonUtils.RetryObject
	for _, v := range failedLogFileMap {
		tempSlice = append(tempSlice, v)
	}
	if len(tempSlice) == 0 {
		tempSlice = []commonUtils.RetryObject{}
	}
	jsonData, err := json.MarshalIndent(tempSlice, "", "  ")
	if err != nil {
		failedLogFile.Close()
		log.Fatal("Error occurred when marshaling to JSON objects: " + err.Error())
	}
	err = failedLogFile.Truncate(0)
	_, err = failedLogFile.WriteAt(jsonData, 0)
	if err != nil {
		failedLogFile.Close()
		log.Fatal("Error occurred when writing to file \"" + failedLogFilename + "\": " + err.Error())
	}
}

func closeFailedLog() error {
	SetToMessageLog()
	log.Println("Local failed log file \"" + failedLogFilename + "\" has closed")
	return failedLogFile.Close()
}
