package g3cmd

import (
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

type NewFlowRequestObject struct {
	Filename string `json:"file_name"`
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
			log.Fatalf("Upload error: %s!\n", err)
			return nil, nil, err
		}
		url = signedURL
	}

	fi, err := file.Stat()
	if err != nil {
		log.Fatal("File Stat Error")
	}

	bar := pb.New(int(fi.Size())).SetUnits(pb.U_BYTES).SetRefreshRate(time.Millisecond).Prefix(fi.Name() + " ")
	req, err := http.NewRequest(http.MethodPut, url, bar.NewProxyReader(file))
	req.ContentLength = fi.Size()

	return req, bar, err
}
