package logs

import (
	"log"
	"os"

	homedir "github.com/mitchellh/go-homedir"
	"github.com/uc-cdis/gen3-client/gen3-client/commonUtils"
)

var MainLogPath string

func Init() {
	homeDir, err := homedir.Dir()
	if err != nil {
		log.Fatalln(err)
	}

	mainPath := homeDir + commonUtils.PathSeparator + ".gen3" + commonUtils.PathSeparator
	if _, err := os.Stat(mainPath); os.IsNotExist(err) { // path to ~/.gen3/logs does not exist
		err = os.Mkdir(mainPath, 0766)
		if err != nil {
			log.Fatal("Cannot create folder \"" + mainPath + "\"")
		}
		log.Println("Created folder \"" + mainPath + "\"")
	}

	MainLogPath = mainPath + "logs" + commonUtils.PathSeparator
	if _, err := os.Stat(MainLogPath); os.IsNotExist(err) { // path to ~/.gen3/logs does not exist
		err = os.Mkdir(MainLogPath, 0766)
		if err != nil {
			log.Fatal("Cannot create folder \"" + MainLogPath + "\"")
		}
		log.Println("Created folder \"" + MainLogPath + "\"")
	}
}

func CloseAll() {
	errorSlice := make([]error, 0)
	errorSlice = append(errorSlice, closeSucceededLog())
	errorSlice = append(errorSlice, closeFailedLog())
	errorSlice = append(errorSlice, closeMessageLog())
	log.SetOutput(os.Stdout)
	for _, err := range errorSlice {
		if err != nil {
			log.Println(err.Error())
		}
	}
}
