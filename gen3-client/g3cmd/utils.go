package g3cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/uc-cdis/gen3-client/gen3-client/jwt"
	pb "gopkg.in/cheggaaa/pb.v1"
)

type ManifestObject struct {
	ObjectID  string `json:"object_id"`
	SubjectID string `json:"subject_id"`
}

type FileUploadRequestObject struct {
	FilePath string
	GUID     string
	Request  *http.Request
}

type PresignedURLRequestObject struct {
	Filename string `json:"file_name"`
}

const FileSizeLimit = 5 * 1024 * 1024 * 1024

func GeneratePresignedURL(filePath string) (string, string, error) {
	request := new(jwt.Request)
	configure := new(jwt.Configure)
	function := new(jwt.Functions)

	function.Config = configure
	function.Request = request

	fileinfo, err := processFilename(filePath)
	if err != nil {
		fmt.Println(err.Error())
	}
	endPointPostfix := "/user/data/upload"
	purObject := PresignedURLRequestObject{Filename: fileinfo.filename}
	objectBytes, err := json.Marshal(purObject)

	respURL, guid, err := function.DoRequestWithSignedHeader(profile, "", endPointPostfix, "application/json", objectBytes)

	if respURL == "" || guid == "" {
		if err != nil {
			return "", "", errors.New("You don't have permission to upload data, detailed error message: " + err.Error())
		}
		return "", "", errors.New("Unknown error has occurred during presigned URL or GUID generation. Please check logs from Gen3 services")
	}
	return respURL, guid, err
}

func GenerateUploadRequest(guid string, url string, file *os.File) (*http.Request, *pb.ProgressBar, error) {
	request := new(jwt.Request)
	configure := new(jwt.Configure)
	function := new(jwt.Functions)

	function.Config = configure
	function.Request = request

	if url == "" {
		endPointPostfix := "/user/data/upload/" + guid
		signedURL, _, err := function.DoRequestWithSignedHeader(profile, "", endPointPostfix, "", nil)
		if err != nil && !strings.Contains(err.Error(), "No GUID found") {
			log.Fatalf("Upload error: %s\n", err)
			return nil, nil, err
		}
		url = signedURL
	}

	fi, err := file.Stat()
	if err != nil {
		log.Fatalf("File stat error for file %s, file may be missing or unreadable because of permissions\n", fi.Name())
	}

	if fi.Size() > FileSizeLimit {
		return nil, nil, errors.New("The file size of file " + fi.Name() + " exceeds the limit allowed and cannot be uploaded. The maximum allowed file size is 5GB.\n")
	}

	bar := pb.New64(fi.Size()).SetUnits(pb.U_BYTES).SetRefreshRate(time.Millisecond * 10).Prefix(fi.Name() + " ")
	pr, pw := io.Pipe()

	go func() {
		var writer io.Writer
		defer pw.Close()
		defer file.Close()

		writer = io.MultiWriter(pw, bar)
		if _, err = io.Copy(writer, file); err != nil {
			log.Fatalf("io.Copy error: %s\n", err)
		}
		if err = pw.Close(); err != nil {
			log.Fatalf("Pipe writer close error: %s\n", err)
		}
	}()

	req, err := http.NewRequest(http.MethodPut, url, pr)
	req.ContentLength = fi.Size()

	return req, bar, err
}
