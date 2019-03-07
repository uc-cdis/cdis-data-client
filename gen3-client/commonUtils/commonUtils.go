package commonUtils

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"

	homedir "github.com/mitchellh/go-homedir"
	pb "gopkg.in/cheggaaa/pb.v1"
)

var PathSeparator = string(os.PathSeparator)

type FileUploadRequestObject struct {
	FilePath     string
	Filename     string
	GUID         string
	PresignedURL string
	Request      *http.Request
	Bar          *pb.ProgressBar
}

type RetryObject struct {
	FilePath     string
	GUID         string
	PresignedURL string
	RetryCount   int
}

func ParseRootPath(filePath string) string {
	if filePath != "" && filePath[0] == '~' {
		homeDir, err := homedir.Dir()
		if err != nil {
			log.Fatalln(err)
		}
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
				isHidden, err := IsHidden(path)
				if err != nil {
					return err
				}
				if !fileInfo.IsDir() && !isHidden {
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
