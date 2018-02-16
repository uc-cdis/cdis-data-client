package jwt

import (
	"bytes"
	"net/http"
	"strings"
)

type APIKeyStruct struct {
	Api_key string
	Key_id  string
}

type AccessTokenStruct struct {
	Access_token string
}

// type UtilInterface interface {
// 	ParseKeyValue(string, string, string) string
// 	ParseConfig(string) Credential
// 	TryReadFile(string) ([]byte, error)
// 	ResponseToString(*http.Response) string
// 	ResponseToBytes(*http.Response) []byte
// }

//type Utils struct{}

func ResponseToString(resp *http.Response) string {
	buf := new(bytes.Buffer)
	buf.ReadFrom(resp.Body)
	return buf.String()
}

func ResponseToBytes(resp *http.Response) []byte {
	strBuf := ResponseToString(resp)
	strBuf = strings.Replace(strBuf, "\n", "", -1)
	return []byte(strBuf)
}
