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

func parse_signature(req *http.Request) string {
	re, _ := regexp.Compile("Signature=(\\S*)")
	return re.FindStringSubmatch(req.Header.Get("Authorization"))[1]
}

func parse_accessKey(req *http.Request) string {
	re, _ := regexp.Compile("Credential=(\\S*?)\\/")
	return re.FindStringSubmatch(req.Header.Get("Authorization"))[1]
}

func parse_SignedHeaders(req *http.Request) []string {
	re, _ := regexp.Compile("SignedHeaders=(\\S*?),")
	return strings.Split(re.FindStringSubmatch(req.Header.Get("Authorization"))[1], ";")
}

func check_expired_time(req_date time.Time) bool {
	end := req_date.Add(time.Minute * time.Duration(15))
	return !req_date.Before(end)
}

func get_exact_request_time(req *http.Request) time.Time {
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

func get_request_scope(req *http.Request, service string) string {
	time := get_exact_request_time(req)
	date := time.Format("20060102")
	return fmt.Sprintf("%v/%v/%v", date, service, BIONIMBUS_REQUEST)
}

func set_req_date(req *http.Request, req_date string) {
	for key, _ := range req.Header {
		if REQUEST_DATE_HEADER == strings.ToLower(key) {
			req.Header.Set(REQUEST_DATE_HEADER, req_date)
		}
	}
}
