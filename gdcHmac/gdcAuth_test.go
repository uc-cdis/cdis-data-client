package gdcHmac

import (
	"net/http"
	"strings"
	"testing"
)

var testMethods = map[string]string{
	"GET":    "67598ed4f2d48da0d554a57b6ddbd395a3382873c5558be855322ce2d02015c1",
	"POST":   "e0ea126a02a9f7ad8741e919031e3e4f3e84124a41a110403f6f871ad45dad01",
	"PUT":    "ee12296d03895236ffb8863161d83624022bacab47e679aa88deb227cce3beb3",
	"DELETE": "e2da8fd17f5e563e1df50d2f7f537444c750b2263e575342536e1ec2fcc0c7b3"}

func TestSigningTasks(t *testing.T) {
	get_vanilla, _ := http.NewRequest("GET", "/", nil)
	get_vanilla.Header.Add("Host", "example.amazonaws.com")
	get_vanilla.Header.Add("X-Amz-Date", "20150830T123600Z")

	meta := new(metadata)
	// (Task 1) The canonical request should be built correctly
	hashedCanonReq := hashedCanonicalRequestV4(get_vanilla, meta)
	if hashedCanonReq != "bb579772317eb040ac9ed261061d46c1f17a8133879d6129b6e1c25292927e63" {
		t.Error("Task 1 error, Canonical Request was misformed!")
	}

	// (Task 2) The string to sign should be built correctly
	stringToSign := stringToSignV4(get_vanilla, hashedCanonReq, meta, "submission")
	if stringToSign != "HMAC-SHA256\n20150830T123600Z\n20150830/submission/bionimbus_request\nbb579772317eb040ac9ed261061d46c1f17a8133879d6129b6e1c25292927e63" {
		t.Error(stringToSign)
		t.Error("Task 2 error")
	}

	// (Task 3) The version 4 signed signature should be correct
	secret_key := "wJalrXUtnFEMI/K7MDENG+bPxRfiCYEXAMPLEKEY"
	signingKey := signingKeyV4(secret_key, meta.date, meta.region, meta.service)
	signature := signatureV4(signingKey, stringToSign)
	if signature != "67598ed4f2d48da0d554a57b6ddbd395a3382873c5558be855322ce2d02015c1" {
		t.Error("Task 3 error, Signature incorrect")
	}
}
func TestSign(t *testing.T) {
	for method, sig := range testMethods {
		get_vanilla, _ := http.NewRequest(method, "/", nil)
		get_vanilla.Header.Add("Host", "example.amazonaws.com")
		get_vanilla.Header.Add("X-Amz-Date", "20150830T123600Z")
		sget_vanilla := Sign(get_vanilla, Credentials{AccessKeyID: "AKIDEXAMPLE", SecretAccessKey: "wJalrXUtnFEMI/K7MDENG+bPxRfiCYEXAMPLEKEY"}, "submission")

		if sget_vanilla.Header.Get("Host") != "example.amazonaws.com" {
			t.Error("Host header incorrect! Expected example.amazonaws.com, got " + sget_vanilla.Header.Get("Host"))
		}
		if sget_vanilla.Header.Get("X-Amz-Date") != "20150830T123600Z" {
			t.Error("X-Amz-Date header incorrect! Expected 20150830T123600Z, got " + sget_vanilla.Header.Get("X-Amz-Date"))
		}
		if sget_vanilla.Header.Get("Authorization") == "" {
			t.Error("Authorization header missing!")
		} else {
			splitAuth := strings.Split(sget_vanilla.Header.Get("Authorization"), " ")
			if splitAuth[0] != "HMAC-SHA256" {
				t.Error("For " + method + " Algorithm in authorization header incorrect! Expected HMAC-SHA256, got " + splitAuth[0])
			}
			if splitAuth[1] != "Credential=AKIDEXAMPLE/20150830/submission/bionimbus_request," {
				t.Error("For " + method + " Credential in authorization header incorrect! Expected Credential=AKIDEXAMPLE/20150830/submission/bionimbus_request,, got " + splitAuth[1])
			}
			if splitAuth[2] != "SignedHeaders=host;x-amz-date," {
				t.Error("For " + method + " SignedHeaders in authorization header incorrect! Expected SignedHeaders=host;x-amz-date,, got " + splitAuth[2])
			}
			if splitAuth[3] != "Signature="+sig {
				t.Error("For " + method + " Signature in authorization header incorrect! Expected Signature=" + sig + ", got " + splitAuth[3])
			}
		}
	}
}

func TestVerify(t *testing.T) {
	for method, sig := range testMethods {
		get_vanilla, _ := http.NewRequest(method, "/", nil)
		get_vanilla.Host = "example.amazonaws.com"
		get_vanilla.Header.Add("User-Agent", "gzip")
		get_vanilla.Header.Add("Authorization", "HMAC-SHA256 Credential=AKIDEXAMPLE/20150830/submission/bionimbus_request, SignedHeaders=host;x-amz-date, Signature="+sig)
		get_vanilla.Header.Add("X-Amz-Date", "20150830T123600Z")
		get_vanilla.Header.Add("Accept-Encoding", "gzip")

		if Verify("submission", get_vanilla, "wJalrXUtnFEMI/K7MDENG+bPxRfiCYEXAMPLEKEY") != true {
			t.Error("For " + method + " Verify was supposed to be true, but is false!")
		}
		if Verify("submission", get_vanilla, "wJalrXUtnFEMI/K7MDENG+bPxRfiCYWRONGWRONG") == true {
			t.Error("For " + method + " Verify was supposed to be false, but is true!")
		}
	}
}
