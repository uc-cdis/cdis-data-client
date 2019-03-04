package logs

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sync"
	"time"
)

var failedLogFilename string
var failedLogFileMap map[string]string
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
	fmt.Println("Local failed log file \"" + failedLogFilename + "\" has opened")

	failedLogFileMap = make(map[string]string)
}

func IsFailedLogMapEmpty() bool {
	return len(failedLogFileMap) == 0
}

func GetFailedLogMap() map[string]string {
	return failedLogFileMap
}

func AddToFailedLogMap(filePath string, presignedUrl string, isMuted bool) {
	failedLogLock.Lock()
	defer failedLogLock.Unlock()
	failedLogFileMap[filePath] = presignedUrl
	if !isMuted {
		fmt.Printf("Failed file entry added for %s\n", filePath)
	}
}

func DeleteFromFailedLogMap(filePath string, isMuted bool) {
	failedLogLock.Lock()
	defer failedLogLock.Unlock()
	delete(failedLogFileMap, filePath)
	if !isMuted {
		fmt.Printf("Failed file entry deleted for %s\n", filePath)
	}
}

func WriteToFailedLog(isMuted bool) {
	failedLogLock.Lock()
	defer failedLogLock.Unlock()
	jsonData, err := json.Marshal(failedLogFileMap)
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
	if !isMuted {
		fmt.Println("Local failed log file updated")
	}
}

func CloseFailedLog() error {
	return failedLogFile.Close()
}
