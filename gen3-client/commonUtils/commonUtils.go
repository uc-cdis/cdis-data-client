package commonUtils

import (
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	homedir "github.com/mitchellh/go-homedir"
	pb "gopkg.in/cheggaaa/pb.v1"
)

// PathSeparator is os dependent path separator char
const PathSeparator = string(os.PathSeparator)

// DefaultTimeout is used to set timeout value for http client
const DefaultTimeout = 30 * time.Second

// FileUploadRequestObject defines a object for file upload
type FileUploadRequestObject struct {
	FilePath     string
	Filename     string
	GUID         string
	PresignedURL string
	Request      *http.Request
	Bar          *pb.ProgressBar
}

// RetryObject defines a object for retry upload
type RetryObject struct {
	FilePath     string
	GUID         string
	PresignedURL string
	RetryCount   int
	Multipart    bool
}

// ParseRootPath parses dirname that has "~" in the beginning
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

// ParseFilePaths generates all possible file paths
func ParseFilePaths(filePath string) ([]string, error) {
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
	log.Println("Finish parsing all file paths for \"" + filePath + "\"")
	return filePaths, err
}
