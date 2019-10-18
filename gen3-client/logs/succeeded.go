package logs

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"os"
	"sync"
)

var succeededLogFilename string
var succeededLogFileMap map[string]string
var succeededLogFile *os.File
var succeededLogLock sync.Mutex

func InitSucceededLog(profile string) {
	succeededLogFilename = MainLogPath + profile + "_succeeded_log.json"

	succeededLogFile, _ = os.OpenFile(succeededLogFilename, os.O_RDWR|os.O_CREATE, 0766)
	fi, err := succeededLogFile.Stat()
	if err != nil {
		succeededLogFile.Close()
		log.Fatal("Error occurred when opening file \"" + succeededLogFilename + "\": " + err.Error())
	}
	log.Println("Local succeeded log file \"" + succeededLogFilename + "\" has opened")

	succeededLogFileMap = make(map[string]string)
	if fi.Size() > 0 {
		data, err := ioutil.ReadAll(succeededLogFile)
		if err != nil {
			succeededLogFile.Close()
			log.Fatal("Error occurred when reading from file \"" + succeededLogFilename + "\": " + err.Error())
		}

		err = json.Unmarshal(data, &succeededLogFileMap)
		if err != nil {
			succeededLogFile.Close()
			log.Fatal("Error occurred when unmarshaling from JSON objects: " + err.Error())
		}
	}
}

func ExistsInSucceededLog(filePath string) bool {
	_, present := succeededLogFileMap[filePath]
	return present
}

func WriteToSucceededLog(filePath string, guid string, isMuted bool) {
	succeededLogLock.Lock()
	defer succeededLogLock.Unlock()
	succeededLogFileMap[filePath] = guid
	jsonData, err := json.MarshalIndent(succeededLogFileMap, "", "  ")
	if err != nil {
		succeededLogFile.Close()
		log.Fatal("Error occurred when marshaling to JSON objects: " + err.Error())
	}
	err = succeededLogFile.Truncate(0)
	_, err = succeededLogFile.WriteAt(jsonData, 0)
	if err != nil {
		succeededLogFile.Close()
		log.Fatal("Error occurred when writing to file \"" + succeededLogFilename + "\": " + err.Error())
	}
	if !isMuted {
		log.Println("Local succeeded log file updated")
	}
}

func closeSucceededLog() error {
	SetToMessageLog()
	log.Println("Local succeeded log file \"" + succeededLogFilename + "\" has closed")
	return succeededLogFile.Close()
}
