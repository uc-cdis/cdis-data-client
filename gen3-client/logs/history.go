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

var HistoryFile string
var HistoryFileMap map[string]string

func InitHistory(profile string) {
	home, err := homedir.Dir()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	historyPath := home + commonUtils.PathSeparator + ".gen3" + commonUtils.PathSeparator

	if _, err := os.Stat(historyPath); os.IsNotExist(err) { // path to ~/.gen3 does not exist
		err = os.Mkdir(historyPath, 0644)
		if err != nil {
			log.Fatal("Cannot create folder \"" + historyPath + "\"")
			os.Exit(1)
		}
		fmt.Println("Created folder \"" + historyPath + "\"")
	}

	HistoryFile = historyPath + profile + "_history.json"

	file, _ := os.OpenFile(HistoryFile, os.O_RDWR|os.O_CREATE, 0644)
	fi, err := file.Stat()
	if err != nil {
		log.Fatal("Error occurred when opening file \"" + HistoryFile + "\": " + err.Error())
	}
	fmt.Println("Local history file \"" + HistoryFile + "\" has opened")

	HistoryFileMap = make(map[string]string)
	if fi.Size() > 0 {
		data, err := ioutil.ReadAll(file)
		if err != nil {
			log.Fatal("Error occurred when reading from file \"" + HistoryFile + "\": " + err.Error())
		}

		err = json.Unmarshal(data, &HistoryFileMap)
		if err != nil {
			log.Fatal("Error occurred when unmarshaling JSON objects: " + err.Error())
		}
	}
}

func ExistsInHistory(filePath string) bool {
	_, present := HistoryFileMap[filePath]
	return present
}

func WriteHistory(filePath string, guid string) {
	writeHistoryFileMap := make(map[string]string)
	writeHistoryFileMap["FilePath"] = filePath
	writeHistoryFileMap["GUID"] = guid
	jsonData, err := json.Marshal(writeHistoryFileMap)
	if err != nil {
		panic(err)
	}
	jsonFile, err := os.OpenFile(HistoryFile, os.O_RDWR|os.O_CREATE, 0666)
	if err != nil {
		panic(err)
	}
	defer jsonFile.Close()

	jsonFile.Write(jsonData)
}
