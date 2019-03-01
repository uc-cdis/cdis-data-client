package logs

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	homedir "github.com/mitchellh/go-homedir"
	"github.com/uc-cdis/gen3-client/gen3-client/commonUtils"
)

var failedLogFilename string
var failedLogFileMap map[string]string
var failedLogFile *os.File
var failedLogLock sync.Mutex

func InitFailedLog(profile string) {
	home, err := homedir.Dir()
	if err != nil {
		log.Fatalln(err)
	}

	failedLogPath := home + commonUtils.PathSeparator + ".gen3" + commonUtils.PathSeparator

	if _, err := os.Stat(failedLogPath); os.IsNotExist(err) { // path to ~/.gen3 does not exist
		err = os.Mkdir(failedLogPath, 0666)
		if err != nil {
			log.Fatal("Cannot create folder \"" + failedLogPath + "\"")
		}
		fmt.Println("Created folder \"" + failedLogPath + "\"")
	}

	failedLogFilename = failedLogPath + profile + "_failed_log_" + time.Now().Format(time.RFC3339) + ".json"

	failedLogFile, err = os.OpenFile(failedLogFilename, os.O_RDWR|os.O_CREATE, 0666)
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
	_, err = failedLogFile.Write(jsonData)
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
