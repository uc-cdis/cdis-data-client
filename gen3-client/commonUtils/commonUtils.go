package commonUtils

import (
	"log"
	"os"
	"os/user"
	"path/filepath"
)

func ParseFilePaths(filePath string) ([]string, error) {
	var fullFilePath string
	if filePath[0] == '~' {
		usr, _ := user.Current()
		homeDir := usr.HomeDir
		fullFilePath = homeDir + filePath[1:]
	} else {
		fullFilePath = filePath
	}

	filePaths, err := filepath.Glob(fullFilePath) // Generating all possible file paths
	for _, filePath := range filePaths {
		file, err := os.Open(filePath)
		if err != nil {
			log.Fatal("File Error")
		}

		if fi, _ := file.Stat(); fi.IsDir() {
			err = filepath.Walk(filePath, func(path string, fileInfo os.FileInfo, err error) error {
				if err != nil {
					return err
				}
				if !fileInfo.IsDir() {
					filePaths = append(filePaths, path)
				}
				return nil
			})
		}
		file.Close()
	}
	return filePaths, err
}
