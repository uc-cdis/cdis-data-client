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

func DecodeJsonFromString(str string, msg Message) error {

	err := json.Unmarshal([]byte(str), &msg)
	if err != nil {
		panic(err)
	}
	return err
}

// func GetUrlFromResponse(resp *http.Response) string {
// 	/*
// 		extract url from http.Response
// 		This function is a replacement for DecodeJsonFromResponse since both Unmarshal and NewDecoder are not stable!
// 	*/
// 	buf := new(bytes.Buffer)
// 	buf.ReadFrom(resp.Body)
// 	data := buf.String()
// 	first := strings.Index(data, "http")
// 	last := strings.LastIndex(data, "\"")

// 	return data[first:last]

// }
