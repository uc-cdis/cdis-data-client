package jwt

//mockgen -destination=mocks/mock_functions.go -package=mocks jwt FunctionInterface

import (
	"bytes"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os/user"
	"path"
	"strconv"
	"strings"

	"github.com/uc-cdis/gen3-client/gen3-client/commonUtils"
)

type Functions struct {
	Request RequestInterface
	Config  ConfigureInterface
}

type FunctionInterface interface {
	DoRequestWithSignedHeader(DoRequest, string, string, string, string, []byte) *http.Response
	ParseFenceURLResponse(*http.Response)
}

type Request struct {
}

type RequestInterface interface {
	MakeARequest(*http.Client, string, string, map[string]string, *bytes.Buffer) (*http.Response, error)
	RequestNewAccessKey(string, *Credential) error
	GetPresignedURL(method string, host *url.URL, endpointPostPrefix string, accessKey string, contentType string, body *bytes.Buffer) *http.Response
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

func (r *Request) RequestNewAccessKey(apiEndpoint string, cred *Credential) error {
	/*
		Request new access token to replace the expired one.

		Args:
			apiEndpoint: the api endpoint for request new access token
		Returns:
			cred: new credential
			err: error

	*/
	body := bytes.NewBufferString("{\"api_key\": \"" + cred.APIKey + "\"}")
	headers := make(map[string]string)
	headers["Content-Type"] = "application/json"
	client := &http.Client{}
	resp, err := r.MakeARequest(client, "POST", apiEndpoint, headers, body)
	var m AccessTokenStruct
	if err != nil {
		return errors.New("Error occurred in RequestNewAccessKey: " + err.Error())
	}

	if resp.StatusCode != 200 {
		if resp.StatusCode == 401 {
			fmt.Println("401 Unauthorized error has occurred! Something went wrong during authentication, please check your configuration and/or credentials.")
		}
		return errors.New("Could not get new access key due to error code " + strconv.Itoa(resp.StatusCode) + ", check fence log for more details.")
	}

	str := ResponseToString(resp)
	err = DecodeJsonFromString(str, &m)
	if err != nil {
		return errors.New("Error occurred in RequestNewAccessKey: " + err.Error())
	}

	if m.Access_token == "" {
		return errors.New("Could not get new access key from response string: " + str)
	}
	cred.AccessKey = m.Access_token
	return nil
}

func (f *Functions) ParseFenceURLResponse(resp *http.Response) (JsonMessage, error) {
	msg := JsonMessage{}

	if resp == nil {
		return msg, errors.New("Nil response received")
	}

	if !(resp.StatusCode == 200 || resp.StatusCode == 201) {
		switch resp.StatusCode {
		case 401:
			return msg, errors.New("401 Unauthorized error has occurred! Something went wrong during authentication, please check your configuration and/or credentials")
		case 403:
			return msg, errors.New("403 Forbidden error has occurred! You don't have premission to access the requested url \"" + resp.Request.URL.String() + "\"")
		case 404:
			return msg, errors.New("Can't find provided GUID at url \"" + resp.Request.URL.String() + "\"")
		case 500:
			return msg, errors.New("500 Internal Server error has occurred! Please try again later")
		case 503:
			return msg, errors.New("503 Service Unavailable error has occurred! Please check backend services for more details")
		default:
			return msg, errors.New("Unexpected server error has occurred! Please check backend services or contact support")
		}
	}

	str := ResponseToString(resp)
	if strings.Contains(str, "Can't find a location for the data") {
		return msg, errors.New("The provided GUID is not found")
	}

	err := DecodeJsonFromString(str, &msg)
	if err != nil {
		return msg, err
	}
	return msg, nil
}

func (f *Functions) DoRequestWithSignedHeader(profile string, configFileType string, endpointPostPrefix string, contentType string, bodyBytes []byte) (string, string, error) {
	/*
		Do request with signed header. User may have more than one profile and use a profile to make a request
	*/
	var resp *http.Response

	cred := f.Config.ParseConfig(profile)
	if cred.APIKey == "" && cred.AccessKey == "" && cred.APIEndpoint == "" {
		return "", "", errors.New("No credentials found in the configuration file! Please use \"./gen3-client configure\" to configure your credentials first")
	}
	host, _ := url.Parse(cred.APIEndpoint)
	prefixEndPoint := host.Scheme + "://" + host.Host
	isExpiredToken := false
	method := "GET"
	if bodyBytes != nil {
		method = "POST"
	}

	if cred.AccessKey != "" {
		resp = f.Request.GetPresignedURL(method, host, endpointPostPrefix, cred.AccessKey, contentType, bytes.NewBuffer(bodyBytes))

		// 401 code is general error code from fence. the error message is also not clear for the case
		// that the token expired. Temporary solution: get new access token and make another attempt.
		if resp.StatusCode == 401 {
			isExpiredToken = true
		} else {
			msg, err := f.ParseFenceURLResponse(resp)
			return msg.Url, msg.GUID, err
		}
	}
	if cred.AccessKey == "" || isExpiredToken {
		err := f.Request.RequestNewAccessKey(prefixEndPoint+"/user/credentials/api/access_token", &cred)
		if err != nil {
			return "", "", err
		}
		usr, err := user.Current()
		homeDir := ""
		if err == nil {
			homeDir = usr.HomeDir
		}
		configPath := path.Join(homeDir + commonUtils.PathSeparator + ".gen3" + commonUtils.PathSeparator + "config")
		content := f.Config.ReadFile(configPath, configFileType)
		f.Config.UpdateConfigFile(cred, []byte(content), cred.APIEndpoint, configPath, profile)

		resp = f.Request.GetPresignedURL(method, host, endpointPostPrefix, cred.AccessKey, contentType, bytes.NewBuffer(bodyBytes))
		msg, err := f.ParseFenceURLResponse(resp)
		return msg.Url, msg.GUID, err
	}
	panic("Unexpected case")
}

func (r *Request) GetPresignedURL(method string, host *url.URL, endpointPostPrefix string, accessKey string, contentType string, body *bytes.Buffer) *http.Response {
	/*
		Get the presigned url
		Args:
			method: either "GET" or "POST"
			host: host endpoint
			endpointPostPrefix: prefix url which is different for download/upload
			accessKey: access key for authZ
			contentType: value of "content-type" HTTP request header
			body: data payload body
		Returns:
			Http response containing presigned url for download and upload
	*/

	apiEndPoint := host.Scheme + "://" + host.Host + endpointPostPrefix
	headers := make(map[string]string)
	headers["Authorization"] = "Bearer " + accessKey
	if contentType != "" {
		headers["Content-Type"] = contentType
	}
	client := &http.Client{}
	resp, err := r.MakeARequest(client, method, apiEndPoint, headers, body)
	if err != nil {
		panic(err)
	}

	return resp
}
