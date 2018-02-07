package gdcHmac

import (
	"encoding/hex"
	"net/http"
	"sort"
	"strings"
	"time"
)

func hashedCanonicalRequestV4(request *http.Request, meta *metadata) string {
	// TASK 1. http://docs.aws.amazon.com/general/latest/gr/sigv4-create-canonical-request.html

	payload := ReadAndReplaceBody(request)
	payloadHash := HashSHA256(payload)
	request.Header.Set("X-Amz-Content-Sha256", payloadHash)
	// Set this in header values to make it appear in the range of headers to sign
	if request.Header.Get("Host") == "" {
		request.Header.Set("Host", request.Host)
	}

	var sortedHeaderKeys []string
	for key, _ := range request.Header {
		switch key {
		case "Content-Type", "Content-Md5", "Host":
		default:
			if !strings.HasPrefix(key, "X-Amz-") {
				continue
			}
		}
		sortedHeaderKeys = append(sortedHeaderKeys, strings.ToLower(key))
	}
	sort.Strings(sortedHeaderKeys)

	var headersToSign string
	for _, key := range sortedHeaderKeys {
		value := strings.TrimSpace(request.Header.Get(key))
		if key == "host" {
			//AWS does not include port in signing request.
			if strings.Contains(value, ":") {
				split := strings.Split(value, ":")
				//port := split[1]
				//if port == "80" || port == "443" {
				value = split[0]
				//}
			}
		}
		headersToSign += key + ":" + value + "\n"
	}

	//payload := ReadAndReplaceBody(request)
	//payloadHash := HashSHA256(payload)
	//request.Header.Set("X-Amz-Content-Sha256", payloadHash)

	meta.signedHeaders = Concat(";", sortedHeaderKeys...)
	canonicalRequest := Concat("\n", request.Method, NormUri(request.URL.Path), NormQuery(request.URL.Query()), headersToSign, meta.signedHeaders, payloadHash)

	return HashSHA256([]byte(canonicalRequest))
}

func stringToSignV4(request *http.Request, hashedCanonReq string, meta *metadata, service string) string {
	// TASK 2. http://docs.aws.amazon.com/general/latest/gr/sigv4-create-string-to-sign.html

	requestTs := request.Header.Get("X-Amz-Date")

	meta.algorithm = ALGORITHM
	meta.service = service
	meta.date = tsDateV4(requestTs)
	meta.credentialScope = GetRequestScope(request, meta.service)

	return Concat("\n", meta.algorithm, requestTs, meta.credentialScope, hashedCanonReq)
}

func signatureV4(signingKey []byte, stringToSign string) string {
	// TASK 3. http://docs.aws.amazon.com/general/latest/gr/sigv4-calculate-signature.html

	return hex.EncodeToString(HmacSHA256(signingKey, stringToSign))
}

func prepareRequestV4(request *http.Request) *http.Request {
	// Do I want this shit in my header?
	necessaryDefaults := map[string]string{
		"Content-Type": "application/x-www-form-urlencoded; charset=utf-8",
		"X-Amz-Date":   timestampV4(),
	}

	for header, value := range necessaryDefaults {
		if request.Header.Get(header) == "" {
			request.Header.Set(header, value)
		}
	}

	if request.URL.Path == "" {
		request.URL.Path += "/"
	}

	return request
}

func signingKeyV4(secretKey, date, service string) []byte {
	kDate := HmacSHA256([]byte("HMAC4"+secretKey), date)
	kService := HmacSHA256(kDate, service)
	kSigning := HmacSHA256(kService, "hmac4_request")
	return kSigning
}

func buildAuthHeaderV4(signature string, meta *metadata, keys Credentials) string {
	credential := keys.AccessKeyID + "/" + meta.credentialScope

	return meta.algorithm +
		" Credential=" + credential +
		", SignedHeaders=" + meta.signedHeaders +
		", Signature=" + signature
}

func timestampV4() string {
	return time.Now().UTC().Format(timeFormatV4)
}

func tsDateV4(timestamp string) string {
	return timestamp[:8]
}

const timeFormatV4 = "20060102T150405Z"
