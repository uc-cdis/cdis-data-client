package gdcHmac

import (
	"io"
	"net/http"
	"net/url"
	"time"
)

func SignedRequest(method string, url_string string, body io.Reader, content_type string, service string, access_key string, secret_key string) (*http.Response, error) {
	uri, err := url.Parse(url_string)
	if err != nil {
		return nil, err
	}

	client := &http.Client{}

	req, err := http.NewRequest(method, url_string, body)
	if err != nil {
		return nil, err
	}
	req.Header.Add("Host", uri.Hostname())
	req.Header.Add("Content-Type", content_type)
	req.Header.Add("X-Amz-Date", time.Now().UTC().Format("20060102T150405Z"))

	signed_req := Sign(req, Credentials{AccessKeyID: access_key, SecretAccessKey: secret_key}, service)

	return client.Do(signed_req)
}
