package logs

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"

	homedir "github.com/mitchellh/go-homedir"
	"github.com/uc-cdis/gen3-client/gen3-client/commonUtils"
)

var succeededLogFilename string
var succeededLogFileMap map[string]string
var succeededLogFile *os.File

func InitSucceededLog(profile string) {
	home, err := homedir.Dir()
	if err != nil {
		log.Fatalln(err)
	}

	succeededLogPath := home + commonUtils.PathSeparator + ".gen3" + commonUtils.PathSeparator

	if _, err := os.Stat(succeededLogPath); os.IsNotExist(err) { // path to ~/.gen3 does not exist
		err = os.Mkdir(succeededLogPath, 0666)
		if err != nil {
			log.Fatal("Cannot create folder \"" + succeededLogPath + "\"")
		}
		fmt.Println("Created folder \"" + succeededLogPath + "\"")
	}

	succeededLogFilename = succeededLogPath + profile + "_succeeded_log.json"

	succeededLogFile, _ = os.OpenFile(succeededLogFilename, os.O_RDWR|os.O_CREATE, 0666)
	fi, err := succeededLogFile.Stat()
	if err != nil {
		succeededLogFile.Close()
		log.Fatal("Error occurred when opening file \"" + succeededLogFilename + "\": " + err.Error())
	}
	fmt.Println("Local succeeded log file \"" + succeededLogFilename + "\" has opened")

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
	tempSucceededLogFileMap := make(map[string]string)
	tempSucceededLogFileMap["FilePath"] = filePath
	tempSucceededLogFileMap["GUID"] = guid
	jsonData, err := json.Marshal(tempSucceededLogFileMap)
	if err != nil {
		succeededLogFile.Close()
		log.Fatal("Error occurred when marshaling to JSON objects: " + err.Error())
	}
	_, err = succeededLogFile.Write(jsonData)
	if err != nil {
		succeededLogFile.Close()
		log.Fatal("Error occurred when writing to file \"" + succeededLogFilename + "\": " + err.Error())
	}
	if !isMuted {
		fmt.Println("Local succeeded log file updated")
	}
}

func CloseSucceededLog() error {
	return succeededLogFile.Close()
}
