package jwt

//go:generate mockgen -destination=mocks/mock_functions.go -package=mocks github.com/uc-cdis/cdis-data-client/jwt FunctionInterface

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"os/user"
	"path"
)

type Functions struct {
	Config  ConfigureInterface
	Request RequestInterface
	Utils   UtilInterface
}

type FunctionInterface interface {
	Requesting(Credential, *url.URL, string) *http.Response
	GetAccessKeyFromFileConfig(string) Credential
	DoRequestWithSignedHeader(DoRequest, string) *http.Response
	SignedRequest(string, string, io.Reader, string) (*http.Response, error)
}

type Request struct {
	Utils UtilInterface
}
type RequestInterface interface {
	MakeARequest(*http.Client, string, string, map[string]string, *bytes.Buffer) (*http.Response, error)
	RequestNewAccessKey(*http.Client, string, *Credential)
}

// func (f *Functions) Set(config Configure, request Request, utils Utils){
// 	f.Config := Configure
// }

func (r *Request) MakeARequest(client *http.Client, method string, path string, headers map[string]string, body *bytes.Buffer) (*http.Response, error) {
	req, err := http.NewRequest(method, path, body)
	if err != nil {
		return nil, err
	}
	for k, v := range headers {
		req.Header.Add(k, v)
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	return resp, nil

}

func (r *Request) RequestNewAccessKey(client *http.Client, path string, cred *Credential) {
	body := bytes.NewBufferString("{\"api_key\": \"" + cred.APIKey + "\"}")
	headers := make(map[string]string)
	headers["Content-Type"] = "application/json"
	resp, err := r.MakeARequest(client, "POST", path, headers, body)
	var m AccessTokenStruct
	if err != nil {
		return
	}
	err = json.Unmarshal(r.Utils.ResponseToBytes(resp), &m)
	if err != nil {
		return
	}
	cred.AccessKey = m.Access_token

}

func (f *Functions) Requesting(cred Credential, host *url.URL, contentType string) *http.Response {
	return &http.Response{}
}

type DoRequest func(cred Credential, host *url.URL, contentType string) *http.Response

var WrapDoConfig = func(u UtilInterface, profile string) Credential {
	return u.ParseConfig(profile)
}

func (f *Functions) GetAccessKeyFromFileConfig(profile string) Credential {
	return f.Utils.ParseConfig(profile)
}

func (f *Functions) DoRequestWithSignedHeader(fn DoRequest, profile string) *http.Response {
	cred := f.GetAccessKeyFromFileConfig(profile)
	if cred.APIKey == "" && cred.AccessKey == "" && cred.APIEndpoint == "" {
		panic("No credential found")
	}

	client := &http.Client{}

	if cred.AccessKey == "" {

		//Include cred.APIKey into the request header to refresh cred.AccessKey then write to profile
		f.Request.RequestNewAccessKey(client, cred.APIEndpoint+"/credentials/cdis/access_token", &cred)

		usr, _ := user.Current()
		homeDir := usr.HomeDir
		configPath := path.Join(homeDir + "/.cdis/config")
		content := f.Config.ReadFile(configPath, "")
		f.Config.UpdateConfigFile(cred, []byte(content), cred.APIEndpoint, configPath, profile)
	}

	contentType := "application/json"
	host, _ := url.Parse(cred.APIEndpoint)
	return fn(cred, host, contentType)

}

func (f *Functions) SignedRequest(method string, url_string string, body io.Reader, access_key string) (*http.Response, error) {

	client := &http.Client{}

	req, err := http.NewRequest(method, url_string, body)
	if err != nil {
		return nil, err
	}
	req.Header.Add("Authorization", "Bearer "+access_key)

	return client.Do(req)
}
