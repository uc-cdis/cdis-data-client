package logs

import (
	"fmt"
	"log"
	"os"

	homedir "github.com/mitchellh/go-homedir"
	"github.com/uc-cdis/gen3-client/gen3-client/commonUtils"
)

var MainLogPath string

func Init() {
	home, err := homedir.Dir()
	if err != nil {
		log.Fatalln(err)
	}

	MainLogPath = home + commonUtils.PathSeparator + ".gen3" + commonUtils.PathSeparator + "logs" + commonUtils.PathSeparator

	if _, err := os.Stat(MainLogPath); os.IsNotExist(err) { // path to ~/.gen3/logs does not exist
		err = os.Mkdir(MainLogPath, 0766)
		if err != nil {
			log.Fatal("Cannot create folder \"" + MainLogPath + "\"")
		}
		fmt.Println("Created folder \"" + MainLogPath + "\"")
	}
}
