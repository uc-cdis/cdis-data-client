package commonUtils

import (
	"bufio"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	homedir "github.com/mitchellh/go-homedir"
	pb "gopkg.in/cheggaaa/pb.v1"
)

// PathSeparator is os dependent path separator char
const PathSeparator = string(os.PathSeparator)

// DefaultTimeout is used to set timeout value for http client
const DefaultTimeout = 120 * time.Second

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
	FilePath   string
	Filename   string
	GUID       string
	RetryCount int
	Multipart  bool
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

// GetAbsolutePath parses input file path to its absolute path and removes the "~" in the beginning
func GetAbsolutePath(filePath string) (string, error) {
	fullFilePath := ParseRootPath(filePath)
	fullFilePath, err := filepath.Abs(fullFilePath)
	return fullFilePath, err
}

// ParseFilePaths generates all possible file paths
func ParseFilePaths(filePath string) ([]string, error) {
	fullFilePath, err := GetAbsolutePath(filePath)
	if err != nil {
		return nil, err
	}
	filePaths, err := filepath.Glob(fullFilePath) // Generating all possible file paths
	if err != nil {
		return nil, err
	}

	for _, filePath := range filePaths {
		func() {
			file, err := os.Open(filePath)
			if err != nil {
				log.Fatal("File error for " + filePath)
			}
			defer file.Close()

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
		}()
	}
	log.Println("Finish parsing all file paths for \"" + filePath + "\"")
	return filePaths, err
}

// AskForConfirmation asks user for confirmation before proceed, will wait if user entered garbage
func AskForConfirmation(s string) bool {
	reader := bufio.NewReader(os.Stdin)

	for {
		fmt.Printf("%s [y/n]: ", s)

		response, err := reader.ReadString('\n')
		if err != nil {
			log.Fatal("Error occurred during parsing user's confirmation: " + err.Error())
		}

		response = strings.ToLower(strings.TrimSpace(response))

		if response == "y" || response == "yes" {
			return true
		} else if response == "n" || response == "no" {
			return false
		}
	}
}
