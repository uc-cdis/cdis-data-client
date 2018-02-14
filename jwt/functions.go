package jwt

import (
	"io"
	"net/http"
)

func SignedRequest(method string, url_string string, body io.Reader, access_key string) (*http.Response, error) {

	client := &http.Client{}

	req, err := http.NewRequest(method, url_string, body)
	if err != nil {
		return nil, err
	}
	req.Header.Add("Authorization", "Bearer "+access_key)

	return client.Do(req)
}
