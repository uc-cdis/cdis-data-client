package gdcHmac

import (
	"net/http"
	"strings"
	"testing"
)

var testMethods = map[string]string{
	"GET":    "61f76cd119862b250b56abe88d00c58cf8a9bf571091313d3a46cbe6093675a7",
	"POST":   "a24aa0b0d658ba1562c1b191a7eebc1c3634b792098f36eb2b811433d2ccd2b1",
	"PUT":    "20f75ef1def6b0fa11ccf4f542fd9628559692edb3670069bd892b45764cac3a",
	"DELETE": "e8496bd6200b3a93380e22c7d56345538ad580a61328163360dc720948e60432"}

func TestSigningTasks(t *testing.T) {
	vanilla_req, _ := http.NewRequest("GET", "/", nil)
	vanilla_req.Header.Add("Host", "example.amazonaws.com")
	vanilla_req.Header.Add("X-Amz-Date", "20150830T123600Z")

	meta := new(metadata)
	// (Task 1) The canonical request should be built correctly
	hashedCanonReq := hashedCanonicalRequestV4(vanilla_req, meta)
	if hashedCanonReq != "bd2af82b09d2569ab8594ef6bcc1638c8675cb753915d0f401b2f40ecde6f823" {
		t.Error("Task 1 error, Canonical Request was misformed!")
	}

	// (Task 2) The string to sign should be built correctly
	stringToSign := stringToSignV4(vanilla_req, hashedCanonReq, meta, "submission")
	if stringToSign != "HMAC-SHA256\n20150830T123600Z\n20150830/submission/bionimbus_request\nbd2af82b09d2569ab8594ef6bcc1638c8675cb753915d0f401b2f40ecde6f823" {
		t.Error(stringToSign)
		t.Error("Task 2 error")
	}

	// (Task 3) The version 4 signed signature should be correct
	secret_key := "wJalrXUtnFEMI/K7MDENG+bPxRfiCYEXAMPLEKEY"
	signingKey := signingKeyV4(secret_key, meta.date, meta.service)
	signature := signatureV4(signingKey, stringToSign)
	if signature != "61f76cd119862b250b56abe88d00c58cf8a9bf571091313d3a46cbe6093675a7" {
		t.Error(signature)
		t.Error("Task 3 error, Signature incorrect")
	}
}
func TestSign(t *testing.T) {
	for method, sig := range testMethods {
		vanilla_req, _ := http.NewRequest(method, "/", nil)
		vanilla_req.Header.Add("Host", "example.amazonaws.com")
		vanilla_req.Header.Add("X-Amz-Date", "20150830T123600Z")
		svanilla_req := Sign(vanilla_req, Credentials{AccessKeyID: "AKIDEXAMPLE", SecretAccessKey: "wJalrXUtnFEMI/K7MDENG+bPxRfiCYEXAMPLEKEY"}, "submission")

		if svanilla_req.Header.Get("Host") != "example.amazonaws.com" {
			t.Error("Host header incorrect! Expected example.amazonaws.com, got " + svanilla_req.Header.Get("Host"))
		}
		if svanilla_req.Header.Get("X-Amz-Date") != "20150830T123600Z" {
			t.Error("X-Amz-Date header incorrect! Expected 20150830T123600Z, got " + svanilla_req.Header.Get("X-Amz-Date"))
		}
		if svanilla_req.Header.Get("Authorization") == "" {
			t.Error("Authorization header missing!")
		} else {
			splitAuth := strings.Split(svanilla_req.Header.Get("Authorization"), " ")
			if splitAuth[0] != "HMAC-SHA256" {
				t.Error("For " + method + ", Algorithm in authorization header incorrect! Expected HMAC-SHA256, got " + splitAuth[0])
			}
			if splitAuth[1] != "Credential=AKIDEXAMPLE/20150830/submission/bionimbus_request," {
				t.Error("For " + method + ", Credential in authorization header incorrect! Expected Credential=AKIDEXAMPLE/20150830/submission/bionimbus_request, got " + splitAuth[1])
			}
			if splitAuth[2] != "SignedHeaders=host;x-amz-content-sha256;x-amz-date," {
				t.Error("For " + method + ", SignedHeaders in authorization header incorrect! Expected SignedHeaders=host;x-amz-date, got " + splitAuth[2])
			}
			if splitAuth[3] != "Signature="+sig {
				t.Error("For " + method + ", Signature in authorization header incorrect! Expected Signature=" + sig + ", got " + splitAuth[3])
			}
		}
	}
}

func TestVerify(t *testing.T) {
	for method, sig := range testMethods {
		vanilla_req, _ := http.NewRequest(method, "/", nil)
		vanilla_req.Host = "example.amazonaws.com"
		vanilla_req.Header.Add("User-Agent", "gzip")
		vanilla_req.Header.Add("Authorization", "HMAC-SHA256 Credential=AKIDEXAMPLE/20150830/submission/bionimbus_request, SignedHeaders=host;x-amz-date, Signature="+sig)
		vanilla_req.Header.Add("X-Amz-Date", "20150830T123600Z")
		vanilla_req.Header.Add("Accept-Encoding", "gzip")

		if Verify("submission", vanilla_req, "wJalrXUtnFEMI/K7MDENG+bPxRfiCYEXAMPLEKEY") != true {
			t.Error("For " + method + " Verify was supposed to be true, but is false!")
		}
		if Verify("submission", vanilla_req, "wJalrXUtnFEMI/K7MDENG+bPxRfiCYWRONGWRONG") == true {
			t.Error("For " + method + " Verify was supposed to be false, but is true!")
		}
	}
}
