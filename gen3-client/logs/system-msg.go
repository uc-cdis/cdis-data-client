package logs

import (
	"io"
	"log"
	"os"
	"sync"
	"time"
)

var messageLogFilename string
var messageLogFile *os.File
var messageLogLock sync.Mutex
var multiWriter io.Writer

func InitMessageLog(profile string) {
	var err error
	messageLogFilename = MainLogPath + profile + "_message_log_" + time.Now().Format("20060102150405MST") + ".log"

	messageLogFile, err = os.OpenFile(messageLogFilename, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0766)
	if err != nil {
		messageLogFile.Close()
		log.Fatalln("Error occurred when opening file \"" + messageLogFilename + "\": " + err.Error())
	}
	multiWriter = io.MultiWriter(os.Stdout, messageLogFile)
	log.SetOutput(multiWriter)
	log.Println("Local failed log file \"" + messageLogFilename + "\" has opened")
}

func closeMessageLog() error {
	log.Println("Local message log file \"" + messageLogFilename + "\" has closed")
	return messageLogFile.Close()
}
