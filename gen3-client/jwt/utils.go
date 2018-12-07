package jwt

import (
	"bytes"
	"encoding/json"
	"net/http"
)

type Message interface{}

type Response interface{}

type AccessTokenStruct struct {
	Access_token string
}

type JsonMessage struct {
	Url  string
	GUID string
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
