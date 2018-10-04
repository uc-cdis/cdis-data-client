package jwt

//mockgen -destination=mocks/mock_functions.go -package=mocks jwt FunctionInterface

import (
	"bytes"
	"errors"
	"io"
	"log"
	"net/http"
	"net/url"
	"os/user"
	"path"
	"strings"
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

	str := ResponseToString(resp)
	DecodeJsonFromString(str, &m)
	if m.Access_token == "" {
		log.Fatalf("Could not get new access key from " + str)
	}

	cred.AccessKey = m.Access_token

}

func (f *Functions) ParseFenceURLResponse(resp *http.Response) (string, error) {
	if resp == nil {
		return "", nil
	}

	msg := JsonMessage{}

	str := ResponseToString(resp)
	if strings.Contains(str, "Can't find a location for the data") {
		return "", errors.New("The provided guid is not found!")
	}

	DecodeJsonFromString(str, &msg)
	if msg.Url == "" {
		return "", errors.New("Can not get url from " + str)
	}

	return msg.Url, nil
}

func (f *Functions) DoRequestWithSignedHeader(profile string, config_file_type string, endpointPostPrefix string) (string, error) {
	/*
		Do request with signed header. User may have more than one profile and use a profile to make a request
	*/

	cred := f.Config.ParseConfig(profile)
	if cred.APIKey == "" && cred.AccessKey == "" && cred.APIEndpoint == "" {
		log.Println("No credential found !!!")
		return "", nil
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
			return f.ParseFenceURLResponse(resp)
		}
	}
	if cred.AccessKey == "" || isExpiredToken {
		f.Request.RequestNewAccessKey(prefixEndPoint+"/user/credentials/cdis/access_token", &cred)
		usr, _ := user.Current()
		configPath := path.Join(usr.HomeDir + "/.gen3/config")
		content := f.Config.ReadFile(configPath, config_file_type)
		f.Config.UpdateConfigFile(cred, []byte(content), cred.APIEndpoint, configPath, profile)
		resp := f.Request.GetPresignedURL(host, endpointPostPrefix, cred.AccessKey)
		return f.ParseFenceURLResponse(resp)
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
	if resp.StatusCode != 200 && resp.StatusCode != 401 && resp.StatusCode != 404 {
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
