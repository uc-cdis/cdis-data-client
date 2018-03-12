package jwt

//mockgen -destination=mocks/mock_functions.go -package=mocks github.com/uc-cdis/cdis-data-client/jwt FunctionInterface

import (
	"bytes"
	"io"
	"log"
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
	DoRequestWithSignedHeader(DoRequest, string, string, string) *http.Response
}

type Request struct {
}
type RequestInterface interface {
	MakeARequest(*http.Client, string, string, map[string]string, *bytes.Buffer) (*http.Response, error)
	RequestNewAccessKey(string, *Credential)
	SignedRequest(string, string, io.Reader, string) (*http.Response, error)
	GetPresignedURL(host *url.URL, endpointPostPrefix string, accessKey string) *http.Response
}

func (r *Request) MakeARequest(client *http.Client, method string, apiEndpoint string, headers map[string]string, body *bytes.Buffer) (*http.Response, error) {
	/*
		Make http request with header and body
	*/
	req, err := http.NewRequest(method, apiEndpoint, body)
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

func (r *Request) RequestNewAccessKey(apiEndpoint string, cred *Credential) {
	/*
		Request new access token to replace the expired one.

		Args:
			apiEndpoint: the api enpoint for request new access token
		Returns:
			cred: new credential

	*/
	body := bytes.NewBufferString("{\"api_key\": \"" + cred.APIKey + "\"}")
	headers := make(map[string]string)
	headers["Content-Type"] = "application/json"
	client := &http.Client{}
	resp, err := r.MakeARequest(client, "POST", apiEndpoint, headers, body)
	var m AccessTokenStruct
	if err != nil {
		panic(err)
	}

	if resp.StatusCode != 200 {
		log.Fatalf("Could not get new access key. " + ResponseToString(resp))
	}

	err = DecodeJsonFromResponse(resp, &m)
	if err != nil {
		log.Fatalf("Could not get url from " + ResponseToString(resp))
	}

	cred.AccessKey = m.Access_token

}

func (f *Functions) DoRequestWithSignedHeader(fn DoRequest, profile string, config_file_type string, endpointPostPrefix string) *http.Response {
	/*
		Do request with signed header. User may have more than one profile and use a profile to make a request
	*/

	cred := f.Config.ParseConfig(profile)
	if cred.APIKey == "" && cred.AccessKey == "" && cred.APIEndpoint == "" {
		panic("No credential found")
	}
	host, _ := url.Parse(cred.APIEndpoint)
	prefixEndPoint := host.Scheme + "://" + host.Host
	isExpiredToken := false

	if cred.AccessKey != "" {
		resp := f.Request.GetPresignedURL(host, endpointPostPrefix, cred.AccessKey)

		// 401 code is general error code from fence. the error message is also not clear for the case
		// that the token expired. Temporary solution: get new access token and make another attempt.
		if resp.StatusCode == 401 {
			isExpiredToken = true
		} else {
			return fn(resp)
		}
	}
	if cred.AccessKey == "" || isExpiredToken {
		f.Request.RequestNewAccessKey(prefixEndPoint+"/user/credentials/cdis/access_token", &cred)
		usr, _ := user.Current()
		configPath := path.Join(usr.HomeDir + "/.cdis/config")
		content := f.Config.ReadFile(configPath, config_file_type)
		f.Config.UpdateConfigFile(cred, []byte(content), cred.APIEndpoint, configPath, profile)
		resp := f.Request.GetPresignedURL(host, endpointPostPrefix, cred.AccessKey)
		return fn(resp)
	}
	panic("Unexpected case")
}

func (r *Request) GetPresignedURL(host *url.URL, endpointPostPrefix string, accessKey string) *http.Response {
	/*
		Get the presigned url
		Args:
			host: host endpoint
			endpointPostPrefix: prefix url which is different for download/upload
			accessKey: access key for authZ
		Returns:
			Http response containing presigned url for download and upload
	*/

	apiEndPoint := host.Scheme + "://" + host.Host + endpointPostPrefix
	resp, err := r.SignedRequest("GET", apiEndPoint, nil, accessKey)
	if err != nil {
		panic(err)
	}
	if resp.StatusCode != 200 && resp.StatusCode != 401 {
		log.Fatalf("Unexpected error %d, %s\n", resp.StatusCode, ResponseToString(resp))
	}

	return resp

}

func (r *Request) SignedRequest(method string, url_string string, body io.Reader, access_key string) (*http.Response, error) {
	/*
		Make a request to server with signed url
		Args:
			method: POST or GET
			url_string: api service endpoint
			body: request body
			access_key: access_key in profile
		Returns:
			http response
	*/
	client := &http.Client{}

	req, err := http.NewRequest(method, url_string, body)
	if err != nil {
		println("error", err)
		return nil, err
	}
	req.Header.Add("Authorization", "Bearer "+access_key)

	return client.Do(req)
}
