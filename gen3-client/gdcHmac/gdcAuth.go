// BASED ON https://github.com/smartystreets/go-aws-auth

// Package awsauth implements modified AWS request signing using Signed Signature Version 4
// to work with GDCAPI
package gdcHmac

import (
	"fmt"
	"net/http"
)

const BIONIMBUS_REQUEST = "bionimbus_request"
const ALGORITHM = "HMAC-SHA256"
const REQUEST_DATE_HEADER = "X-Amz-Date"
const HASHED_REQUEST_CONTENT = "x-amz-content-sha256"
const REQUEST_HEADER_PREFIX = "x-amz-"
const AUTHORIZATION_HEADER = "Authorization"
const CLIENT_CONTEXT_HEADER = "x-amz-client-context"

type Credentials struct {
	AccessKeyID     string
	SecretAccessKey string
}

// Sign signs a request with Signed Signature Version 4.
// TRYING TO COPY PYTHON SIGN_AUTH
func Sign(request *http.Request, credentials Credentials, service string) *http.Request {
	secret_key := credentials.SecretAccessKey
	if request.URL.Path == "" {
		request.URL.Path += "/"
	}

	meta := new(metadata)

	// Task 1
	hashedCanonReq := hashedCanonicalRequestV4(request, meta)

	// Task 2
	stringToSign := stringToSignV4(request, hashedCanonReq, meta, service)

	// Task 3
	signingKey := signingKeyV4(secret_key, meta.date, meta.service)

	signature := signatureV4(signingKey, stringToSign)

	request.Header.Set("Authorization", buildAuthHeaderV4(signature, meta, credentials))

	return request
}

func Verify(service string, req *http.Request, secret_key string) bool {
	signature := ParseSignature(req)
	accessKey := ParseAccessKey(req)
	SignedHeaders := ParseSignedHeaders(req)
	credentials := Credentials{AccessKeyID: accessKey, SecretAccessKey: secret_key}
	req_time := GetExactRequestTime(req)
	if CheckExpiredTime(req_time) {
		fmt.Println("Request expired")
		return false
	}
	original_req, _ := http.NewRequest(req.Method, req.URL.String(), req.Body)
	for _, v := range SignedHeaders {
		if v == "host" {
			original_req.Header.Add("Host", req.Host)
		} else {
			original_req.Header.Add(v, req.Header.Get(v))
		}
	}

	sReq := Sign(original_req, credentials, service)
	if ParseSignature(sReq) == signature {
		return true
	} else {
		return false
	}
}
