package commonUtils

import (
	"fmt"
	"log"
	"os"
	"os/user"
	"path/filepath"
)

func ParseRootPath(filePath string) string {
	if filePath != "" && filePath[0] == '~' {
		usr, _ := user.Current()
		homeDir := usr.HomeDir
		return homeDir + filePath[1:]
	}
	return filePath
}

func ParseFilePaths(filePath string) ([]string, error) {
	fmt.Println("\nBegin parsing all file paths for \"" + filePath + "\"")
	fullFilePath := ParseRootPath(filePath)
	filePaths, err := filepath.Glob(fullFilePath) // Generating all possible file paths

	for _, filePath := range filePaths {
		file, err := os.Open(filePath)
		if err != nil {
			log.Fatal("File error for " + filePath)
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
	fmt.Println("Finish parsing all file paths for \"" + filePath + "\"")
	return filePaths, err
}
