package jwt

//go:generate mockgen -destination=mocks/mock_functions.go -package=mocks github.com/uc-cdis/cdis-data-client/jwt FunctionInterface

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"os/user"
	"path"
)

type Functions struct {
	Request RequestInterface
	Config  ConfigureInterface
}

type FunctionInterface interface {
	Requesting(Credential, *url.URL, string) (*http.Response, error)
	DoRequestWithSignedHeader(DoRequest, string, string) (*http.Response, error)
	SignedRequest(string, string, io.Reader, string) (*http.Response, error)
}

type Request struct {
}
type RequestInterface interface {
	MakeARequest(*http.Client, string, string, map[string]string, *bytes.Buffer) (*http.Response, error)
	RequestNewAccessKey(string, *Credential)
}

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

func (r *Request) RequestNewAccessKey(path string, cred *Credential) {
	body := bytes.NewBufferString("{\"api_key\": \"" + cred.APIKey + "\"}")
	headers := make(map[string]string)
	headers["Content-Type"] = "application/json"
	client := &http.Client{}
	resp, err := r.MakeARequest(client, "POST", path, headers, body)
	var m AccessTokenStruct
	if err != nil {
		return
	}
	//println(ResponseToString(resp))
	err = json.Unmarshal(ResponseToBytes(resp), &m)
	if err != nil {
		return
	}
	cred.AccessKey = m.Access_token
}

func (f *Functions) Requesting(cred Credential, host *url.URL, contentType string) (*http.Response, error) {
	return &http.Response{}, nil
}

type DoRequest func(cred Credential, host *url.URL, contentType string) (*http.Response, error)

func (f *Functions) DoRequestWithSignedHeader(fn DoRequest, profile string, file_type string) (*http.Response, error) {
	cred := f.Config.ParseConfig(profile)
	if cred.APIKey == "" && cred.AccessKey == "" && cred.APIEndpoint == "" {
		panic("No credential found")
	}
	contentType := "application/json"
	host, _ := url.Parse(cred.APIEndpoint)
	isExpiredToken := false

	if cred.AccessKey != "" {
		resp, err := fn(cred, host, contentType)
		if resp.StatusCode == 401 {
			isExpiredToken = true
		} else {
			return resp, err
		}
	}
	if cred.AccessKey == "" || isExpiredToken {
		f.Request.RequestNewAccessKey(cred.APIEndpoint+"/credentials/cdis/access_token", &cred)
		usr, _ := user.Current()
		configPath := path.Join(usr.HomeDir + "/.cdis/config")
		content := f.Config.ReadFile(configPath, file_type)
		f.Config.UpdateConfigFile(cred, []byte(content), cred.APIEndpoint, configPath, profile)
		return fn(cred, host, contentType)
	}
	return nil,  error(errors.New("Unexpected case"))
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
