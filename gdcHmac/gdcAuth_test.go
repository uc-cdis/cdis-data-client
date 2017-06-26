package gdcHmac

import (
	"bytes"
	"fmt"
	"net/http"
	"testing"
)

/*
func TestSign(t *testing.T) {
	secret_key := "wJalrXUtnFEMI/K7MDENG+bPxRfiCYEXAMPLEKEY"
	service := "submission"
	date := "20110909"
	//key := HMAC4SigningKey(secret_key, service, date)
	//b := bytes.NewBufferString("Action=ListUsers&Version=2010-05-08")
	req, _ := http.NewRequest("POST", "", nil)
	req.Header.Add("Host", "example.amazonaws.com")
	req.Header.Add("X-Amz-Date", "20150830T123600Z")

	//auth = HMAC4Auth("dummy", key)

	sreq := Sign(req, Credentials{AccessKeyID: "", SecretAccessKey: secret_key}, service, date)
	signature := sreq.Header.Get("Authorization")

	expected := ""
	if signature != expected {
		t.Errorf("Signature incorrect. got: %v, want: %v", signature, expected)
	}

}
*/

func TestSet_req_date(t *testing.T) {
	req, _ := http.NewRequest("POST", "", nil)
	req.Header.Add("Host", "example.amazonaws.com")
	req.Header.Add("X-Amz-Date", "20150830T123600Z")
	date := "20110909"
	//fmt.Println(req.Header)
	set_req_date(req, date)
	//fmt.Println(req.Header)
}

func TestHashBody(t *testing.T) {
	req, _ := http.NewRequest("POST", "", nil)
	req.Header.Add("Host", "example.amazonaws.com")
	req.Header.Add("X-Amz-Date", "20150830T123600Z")
	//fmt.Println(req.Header)

	payload := readAndReplaceBody(req)
	//payloadHash :=
	req.Header.Set("X-Amz-Content-Sha256", hashSHA256(payload))
	//fmt.Println(req.Header)
}

func TestHashedCanonicalRequestV4(t *testing.T) {
	req, _ := http.NewRequest("POST", "", nil)
	req.Header.Add("Host", "example.amazonaws.com")
	req.Header.Add("X-Amz-Date", "20110909T233600Z")
	date := "20110909"
	set_req_date(req, date)

	//meta := new(metadata)
	//hashedCanonReq :=
	//fmt.Println(hashedCanonicalRequestV4(req, meta))
}

func TestSign(t *testing.T) {
	body := "ExampleBody"
	buf := bytes.NewBufferString(body)
	req, _ := http.NewRequest("POST", "", buf)
	req.Header.Add("Host", "example.amazonaws.com")
	req.Header.Add("X-Amz-Date", "20110909T233600Z")
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded; charset=utf-8")
	fmt.Println("Initial req.Header")
	fmt.Println(req.Header)
	date := "20110909"
	set_req_date(req, date)

	signed_req := Sign(req, Credentials{AccessKeyID: "20150830/us-east-1/service", SecretAccessKey: "wJalrXUtnFEMI/K7MDENG+bPxRfiCYEXAMPLEKEY"}, "submission", date)
	fmt.Println(signed_req.Header)
	//hashedCanonReq :=
	//fmt.Println(hashedCanonicalRequestV4(req, meta))
}
