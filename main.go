package main

import (
	"github.com/uc-cdis/cdis-data-client/jwt"
)

func main() {
	//cmd.Execute()
	//jwt.DoRequestWithSignedHeader(jwt.Requesting, "default")
	//test := jwt.Functions{Config: jwt.Configure{}, Request: jwt.Request{}, Utils: jwt.Utils{}}

	utils := new(jwt.Utils)
	request := new(jwt.Request)
	request.Utils = utils

	test := jwt.Functions{Config: new(jwt.Configure), Request: request, Utils: utils}

	test.DoRequestWithSignedHeader(test.Requesting, "default")
}
