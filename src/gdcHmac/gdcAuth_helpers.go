package gdcHmac

import (
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"
)

type metadata struct {
	algorithm       string
	credentialScope string
	signedHeaders   string
	date            string
	region          string
	service         string
}

func ParseSignature(req *http.Request) string {
	re, _ := regexp.Compile("Signature=(\\S*)")
	return re.FindStringSubmatch(req.Header.Get("Authorization"))[1]
}

func ParseAccessKey(req *http.Request) string {
	re, _ := regexp.Compile("Credential=(\\S*?)\\/")
	return re.FindStringSubmatch(req.Header.Get("Authorization"))[1]
}

func ParseSignedHeaders(req *http.Request) []string {
	re, _ := regexp.Compile("SignedHeaders=(\\S*?),")
	return strings.Split(re.FindStringSubmatch(req.Header.Get("Authorization"))[1], ";")
}

func CheckExpiredTime(req_date time.Time) bool {
	end := req_date.Add(time.Minute * time.Duration(15))
	return !req_date.Before(end)
}

func GetExactRequestTime(req *http.Request) time.Time {
	layout1 := "20060102T150405Z"
	layout2 := "20060102"
	for key, _ := range req.Header {
		if key == "X-Amz-Date" {
			if len(req.Header.Get(key)) == 8 {
				t, _ := time.Parse(layout2, req.Header.Get(key))
				return t
			}
			t, _ := time.Parse(layout1, req.Header.Get(key))
			return t
		}
	}
	for key, _ := range req.Header {
		if key == "date" {
			if len(req.Header.Get(key)) == 8 {
				t, _ := time.Parse(layout2, req.Header.Get(key))
				return t
			}
			t, _ := time.Parse(layout1, req.Header.Get(key))
			return t
		}
	}
	fmt.Println("Misformed header: No \"X-Amz-Date\" or \"date\" header")
	return time.Now()
}

func GetRequestScope(req *http.Request, service string) string {
	requestTime := GetExactRequestTime(req)
	date := requestTime.Format("20060102")
	return fmt.Sprintf("%v/%v/%v", date, service, BIONIMBUS_REQUEST)
}
