package jwt

import (
	"bytes"
	"encoding/json"
	"net/http"
)

type Message interface{}

type AccessTokenStruct struct {
	Access_token string
}

type JsonMessage struct {
	Url string
}

type DoRequest func(*http.Response) *http.Response

func ResponseToString(resp *http.Response) string {
	buf := new(bytes.Buffer)
	buf.ReadFrom(resp.Body)
	return buf.String()
}

func DecodeJsonFromResponse(resp *http.Response, msg Message) error {
	err := json.NewDecoder(resp.Body).Decode(&msg)
	if err != nil {
		panic(err)
	}
	return err
}
