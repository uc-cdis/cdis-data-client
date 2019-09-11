package commonUtils

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	homedir "github.com/mitchellh/go-homedir"
	pb "gopkg.in/cheggaaa/pb.v1"
)

// IndexdIndexEndpoint is the endpoint postfix for INDEXD index
const IndexdIndexEndpoint = "/index/index"

// FenceUserEndpoint is the endpoint postfix for FENCE user
const FenceUserEndpoint = "/user/user"

// FenceDataEndpoint is the endpoint postfix for FENCE data
const FenceDataEndpoint = "/user/data"

// FenceDataUploadEndpoint is the endpoint postfix for FENCE data upload
const FenceDataUploadEndpoint = FenceDataEndpoint + "/upload"

// FenceDataDownloadEndpoint is the endpoint postfix for FENCE data download
const FenceDataDownloadEndpoint = FenceDataEndpoint + "/download"

// FenceDataMultipartInitEndpoint is the endpoint postfix for FENCE multipart init
const FenceDataMultipartInitEndpoint = FenceDataEndpoint + "/multipart/init"

// FenceDataMultipartUploadEndpoint is the endpoint postfix for FENCE multipart upload
const FenceDataMultipartUploadEndpoint = FenceDataEndpoint + "/multipart/upload"

// FenceDataMultipartCompleteEndpoint is the endpoint postfix for FENCE multipart complete
const FenceDataMultipartCompleteEndpoint = FenceDataEndpoint + "/multipart/complete"

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

// FileDownloadResponseObject defines a object for file download
type FileDownloadResponseObject struct {
	DownloadPath string
	Filename     string
	GUID         string
	URL          string
	Range        int64
	Overwrite    bool
	Skip         bool
	Response     *http.Response
	Writer       io.Writer
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

	filePaths = cleanupHiddenFiles(filePaths)

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
					} else if isHidden {
						log.Printf("File %s is a hidden file and will be skipped\n", path)
					}
					return nil
				})
			}
		}()
	}
	log.Println("Finish parsing all file paths for \"" + fullFilePath + "\"")
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

func cleanupHiddenFiles(filePaths []string) []string {
	i := 0
	for _, filePath := range filePaths {
		isHidden, err := IsHidden(filePath)
		if err != nil {
			log.Println("Error occurred when checking hidden files: " + err.Error())
			continue
		}

		if isHidden {
			log.Printf("File %s is a hidden file and will be skipped\n", filePath)
			continue
		}
		filePaths[i] = filePath
		i++
	}
	return filePaths[:i]
}
