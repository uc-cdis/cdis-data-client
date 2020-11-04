package jwt

import (
	"bytes"
	"encoding/json"
	"net/http"
)

type Message interface{}

type Response interface{}

type AccessTokenStruct struct {
	AccessToken string `json:"access_token"`
}

type JsonMessage struct {
	URL          string   `json:"url"`
	GUID         string   `json:"guid"`
	UploadID     string   `json:"uploadId"`
	PresignedURL string   `json:"presigned_url"`
	FileName     string   `json:"file_name"`
	URLs         []string `json:"urls"`
	Size         int64    `json:"size"`
}

type DoRequest func(*http.Response) *http.Response

func ResponseToString(resp *http.Response) string {
	buf := new(bytes.Buffer)
	buf.ReadFrom(resp.Body)
	return buf.String()
}

func DecodeJsonFromString(str string, msg Message) error {
	err := json.Unmarshal([]byte(str), &msg)
	return err
}
