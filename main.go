package main

import (
	"net/http"

	"github.com/uc-cdis/cdis-data-client/jwt"
)

func Requesting(*http.Response) *http.Response {
	return &http.Response{}
}
func main() {
	//cmd.Execute()
	test := jwt.Functions{Config: new(jwt.Configure), Request: new(jwt.Request)}
	test.DoRequestWithSignedHeader(Requesting, "default", "text", "/data")
}
