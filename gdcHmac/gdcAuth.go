// BASED ON https://github.com/smartystreets/go-aws-auth

// Package awsauth implements modified AWS request signing using Signed Signature Version 4
// to work with GDCAPI
package gdcHmac

import (
	"fmt"
	"net/http"
	"time"
)

const BIONIMBUS_REQUEST = "bionimbus_request"
const ALGORITHM = "HMAC-SHA256"
const REQUEST_DATE_HEADER = "x-amz-date"
const HASHED_REQUEST_CONTENT = "x-amz-content-sha256"
const REQUEST_HEADER_PREFIX = "x-amz-"
const AUTHORIZATION_HEADER = "Authorization"
const CLIENT_CONTEXT_HEADER = "x-amz-client-context"

type Credentials struct {
	AccessKeyID     string
	SecretAccessKey string
	SecurityToken   string `json:"Token"`
	Expiration      time.Time
}

// Sign signs a request with Signed Signature Version 4.
// TRYING TO COPY PYTHON SIGN_AUTH
func Sign(request *http.Request, credentials Credentials, service string, req_date string) *http.Request {
	secret_key := credentials.SecretAccessKey
	if request.URL.Path == "" {
		request.URL.Path += "/"
	}

	set_req_date(request, req_date)

	meta := new(metadata)

	// Task 1
	hashedCanonReq := hashedCanonicalRequestV4(request, meta)

	// Task 2
	stringToSign := stringToSignV4(request, hashedCanonReq, meta, service)

	// Task 3
	signingKey := signingKeyV4(secret_key, meta.date, meta.region, meta.service)
	signature := signatureV4(signingKey, stringToSign)

	request.Header.Set("Authorization", buildAuthHeaderV4(signature, meta, credentials))

	return request
}

func Verify(service string, req *http.Request, secret_key string) bool {
	signature := parse_signature(req)
	accessKey := parse_accessKey(req)
	credentials := Credentials{AccessKeyID: accessKey, SecretAccessKey: secret_key}
	req_time := get_exact_request_time(req)
	req_date := req_time.Format("20060102")
	if check_expired_time(req_time) {
		fmt.Println("Request expired")
		return false
	}
	sReq := Sign(req, credentials, service, req_date)
	if parse_signature(sReq) == signature {
		return true
	} else {
		return false
	}
}
