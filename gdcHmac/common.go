package gdcHmac

import (
	"bytes"
	"crypto/hmac"
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type location struct {
	ec2     bool
	checked bool
}

var loc *location

// serviceAndRegion parsers a hostname to find out which ones it is.
// http://docs.aws.amazon.com/general/latest/gr/rande.html
func serviceAndRegion(host string) (service string, region string) {
	// These are the defaults if the hostname doesn't suggest something else
	region = "us-east-1"
	service = "s3"

	parts := strings.Split(host, ".")
	if len(parts) == 4 {
		// Either service.region.amazonaws.com or virtual-host.region.amazonaws.com
		if parts[1] == "s3" {
			service = "s3"
		} else if strings.HasPrefix(parts[1], "s3-") {
			region = parts[1][3:]
			service = "s3"
		} else {
			service = parts[0]
			region = parts[1]
		}
	} else if len(parts) == 5 {
		service = parts[2]
		region = parts[1]
	} else {
		// Either service.amazonaws.com or s3-region.amazonaws.com
		if strings.HasPrefix(parts[0], "s3-") {
			region = parts[0][3:]
		} else {
			service = parts[0]
		}
	}

	if region == "external-1" {
		region = "us-east-1"
	}

	return
}

func augmentRequestQuery(request *http.Request, values url.Values) *http.Request {
	for key, array := range request.URL.Query() {
		for _, value := range array {
			values.Set(key, value)
		}
	}

	request.URL.RawQuery = values.Encode()

	return request
}

func hmacSHA256(key []byte, content string) []byte {
	mac := hmac.New(sha256.New, key)
	mac.Write([]byte(content))
	return mac.Sum(nil)
}

func hmacSHA1(key []byte, content string) []byte {
	mac := hmac.New(sha1.New, key)
	mac.Write([]byte(content))
	return mac.Sum(nil)
}

func hashSHA256(content []byte) string {
	h := sha256.New()
	h.Write(content)
	return fmt.Sprintf("%x", h.Sum(nil))
}

func hashMD5(content []byte) string {
	h := md5.New()
	h.Write(content)
	return base64.StdEncoding.EncodeToString(h.Sum(nil))
}

func readAndReplaceBody(request *http.Request) []byte {
	if request.Body == nil {
		return []byte{}
	}
	payload, _ := ioutil.ReadAll(request.Body)
	request.Body = ioutil.NopCloser(bytes.NewReader(payload))
	return payload
}

func concat(delim string, str ...string) string {
	return strings.Join(str, delim)
}

var now = func() time.Time {
	return time.Now().UTC()
}

func normuri(uri string) string {
	parts := strings.Split(uri, "/")
	for i := range parts {
		parts[i] = encodePathFrag(parts[i])
	}
	return strings.Join(parts, "/")
}

func encodePathFrag(s string) string {
	hexCount := 0
	for i := 0; i < len(s); i++ {
		c := s[i]
		if shouldEscape(c) {
			hexCount++
		}
	}
	t := make([]byte, len(s)+2*hexCount)
	j := 0
	for i := 0; i < len(s); i++ {
		c := s[i]
		if shouldEscape(c) {
			t[j] = '%'
			t[j+1] = "0123456789ABCDEF"[c>>4]
			t[j+2] = "0123456789ABCDEF"[c&15]
			j += 3
		} else {
			t[j] = c
			j++
		}
	}
	return string(t)
}

func shouldEscape(c byte) bool {
	if 'a' <= c && c <= 'z' || 'A' <= c && c <= 'Z' {
		return false
	}
	if '0' <= c && c <= '9' {
		return false
	}
	if c == '-' || c == '_' || c == '.' || c == '~' {
		return false
	}
	return true
}

func normquery(v url.Values) string {
	queryString := v.Encode()

	// Go encodes a space as '+' but Amazon requires '%20'. Luckily any '+' in the
	// original query string has been percent escaped so all '+' chars that are left
	// were originally spaces.

	return strings.Replace(queryString, "+", "%20", -1)
}
