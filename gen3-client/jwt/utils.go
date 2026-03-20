package jwt

import (
	"bytes"
	"encoding/json"
	"log"
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
	Rev          string   `json:"rev"`
}

type DoRequest func(*http.Response) *http.Response

func ResponseToString(resp *http.Response) string {
	buf := new(bytes.Buffer)
	_, err := buf.ReadFrom(resp.Body)
	if err != nil {
		log.Println("WARNING: unable to parse response body")
		return ""
	}
	return buf.String()
}

func DecodeJsonFromString(str string, msg Message) error {
	err := json.Unmarshal([]byte(str), &msg)
	return err
}
