package main

import (
	"github.com/uc-cdis/cdis-data-client/jwt"
)

func main() {
	//cmd.Execute()
	//jwt.DoRequestWithSignedHeader(jwt.Requesting, "default")
	//test := jwt.Functions{Config: jwt.Configure{}, Request: jwt.Request{}, Utils: jwt.Utils{}}

	test := jwt.Functions{Config: new(jwt.Configure), Request: new(jwt.Request), Utils: new(jwt.Utils)}
	test.DoRequestWithSignedHeader(test.Requesting, "default")
}
