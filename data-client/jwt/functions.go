package jwt

//go:generate mockgen -destination=./data-client/mocks/mock_functions.go -package=mocks github.com/calypr/data-client/data-client/jwt FunctionInterface
//go:generate mockgen -destination=./data-client/mocks/mock_request.go -package=mocks github.com/calypr/data-client/data-client/jwt RequestInterface

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/calypr/data-client/data-client/commonUtils"
	"github.com/hashicorp/go-version"
)

type Functions struct {
	Request RequestInterface
	Config  ConfigureInterface
}

type FunctionInterface interface {
	CheckPrivileges(profileConfig *Credential) (string, map[string]interface{}, error)
	CheckForShepherdAPI(profileConfig *Credential) (bool, error)
	GetResponse(profileConfig *Credential, endpointPostPrefix string, method string, contentType string, bodyBytes []byte) (string, *http.Response, error)
	DoRequestWithSignedHeader(profileConfig *Credential, endpointPostPrefix string, contentType string, bodyBytes []byte) (JsonMessage, error)
	ParseFenceURLResponse(resp *http.Response) (JsonMessage, error)
	GetHost(profileConfig *Credential) (*url.URL, error)
}

type Request struct {
}

type RequestInterface interface {
	MakeARequest(method string, apiEndpoint string, accessToken string, contentType string, headers map[string]string, body *bytes.Buffer, noTimeout bool) (*http.Response, error)
	RequestNewAccessToken(accessTokenEndpoint string, profileConfig *Credential) error
}

func (r *Request) MakeARequest(method string, apiEndpoint string, accessToken string, contentType string, headers map[string]string, body *bytes.Buffer, noTimeout bool) (*http.Response, error) {
	/*
		Make http request with header and body
	*/
	if headers == nil {
		headers = make(map[string]string)
	}
	if accessToken != "" {
		headers["Authorization"] = "Bearer " + accessToken
	}
	if contentType != "" {
		headers["Content-Type"] = contentType
	}
	var client *http.Client
	if noTimeout {
		client = &http.Client{}
	} else {
		client = &http.Client{Timeout: commonUtils.DefaultTimeout}
	}
	var req *http.Request
	var err error
	if body == nil {
		req, err = http.NewRequest(method, apiEndpoint, nil)
	} else {
		req, err = http.NewRequest(method, apiEndpoint, body)
	}

	if err != nil {
		return nil, errors.New("Error occurred during generating HTTP request: " + err.Error())
	}
	for k, v := range headers {
		req.Header.Add(k, v)
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, errors.New("Error occurred during making HTTP request: " + err.Error())
	}
	return resp, nil
}

func (r *Request) RequestNewAccessToken(accessTokenEndpoint string, profileConfig *Credential) error {
	/*
		Request new access token to replace the expired one.

		Args:
			accessTokenEndpoint: the api endpoint for request new access token
		Returns:
			profileConfig: new credential
			err: error

	*/
	body := bytes.NewBufferString("{\"api_key\": \"" + profileConfig.APIKey + "\"}")
	resp, err := r.MakeARequest("POST", accessTokenEndpoint, "", "application/json", nil, body, false)
	var m AccessTokenStruct
	// parse resp error codes first for profile configuration verification
	if resp != nil && resp.StatusCode != 200 {
		return errors.New("Error occurred in RequestNewAccessToken with error code " + strconv.Itoa(resp.StatusCode) + ", check FENCE log for more details.")
	}
	if err != nil {
		return errors.New("Error occurred in RequestNewAccessToken: " + err.Error())
	}
	defer resp.Body.Close()

	str := ResponseToString(resp)
	err = DecodeJsonFromString(str, &m)
	if err != nil {
		return errors.New("Error occurred in RequestNewAccessToken: " + err.Error())
	}

	if m.AccessToken == "" {
		return errors.New("Could not get new access key from response string: " + str)
	}
	profileConfig.AccessToken = m.AccessToken
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
			return msg, errors.New("403 Forbidden error has occurred! You don't have permission to access the requested url \"" + resp.Request.URL.String() + "\"")
		case 404:
			return msg, errors.New("404 Not found error has occurred! The requested url \"" + resp.Request.URL.String() + "\" cannot be found or one of the requested resources cannot be found")
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

func (f *Functions) CheckForShepherdAPI(profileConfig *Credential) (bool, error) {
	// Check if Shepherd is enabled
	if profileConfig.UseShepherd == "false" {
		return false, nil
	}
	if profileConfig.UseShepherd != "true" && commonUtils.DefaultUseShepherd == false {
		return false, nil
	}
	// If Shepherd is enabled, make sure that the commons has a compatible version of Shepherd deployed.
	// Compare the version returned from the Shepherd version endpoint with the minimum acceptable Shepherd version.
	var minShepherdVersion string
	if profileConfig.MinShepherdVersion == "" {
		minShepherdVersion = commonUtils.DefaultMinShepherdVersion
	} else {
		minShepherdVersion = profileConfig.MinShepherdVersion
	}

	_, res, err := f.GetResponse(profileConfig, commonUtils.ShepherdVersionEndpoint, "GET", "", nil)
	if err != nil {
		return false, errors.New("Error occurred during generating HTTP request: " + err.Error())
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		return false, nil
	}
	bodyBytes, err := io.ReadAll(res.Body)
	if err != nil {
		return false, errors.New("Error occurred when reading HTTP request: " + err.Error())
	}
	body, err := strconv.Unquote(string(bodyBytes))
	if err != nil {
		return false, fmt.Errorf("Error occurred when parsing version from Shepherd: %v: %v", string(body), err)
	}
	// Compare the version in the response to the target version
	ver, err := version.NewVersion(body)
	if err != nil {
		return false, fmt.Errorf("Error occurred when parsing version from Shepherd: %v: %v", string(body), err)
	}
	minVer, err := version.NewVersion(minShepherdVersion)
	if err != nil {
		return false, fmt.Errorf("Error occurred when parsing minimum acceptable Shepherd version: %v: %v", minShepherdVersion, err)
	}
	if ver.GreaterThanOrEqual(minVer) {
		return true, nil
	}
	return false, fmt.Errorf("Shepherd is enabled, but %v does not have correct Shepherd version. (Need Shepherd version >=%v, got %v)", profileConfig.APIEndpoint, minVer, ver)
}

func (f *Functions) GetResponse(profileConfig *Credential, endpointPostPrefix string, method string, contentType string, bodyBytes []byte) (string, *http.Response, error) {

	var resp *http.Response
	var err error

	if profileConfig.APIKey == "" && profileConfig.AccessToken == "" && profileConfig.APIEndpoint == "" {
		return "", resp, errors.New(fmt.Sprintf("No credentials found in the configuration file! Please use \"./data-client configure\" to configure your credentials first %s", profileConfig))
	}
	host, _ := url.Parse(profileConfig.APIEndpoint)
	prefixEndPoint := host.Scheme + "://" + host.Host
	apiEndpoint := host.Scheme + "://" + host.Host + endpointPostPrefix
	isExpiredToken := false
	if profileConfig.AccessToken != "" {
		resp, err = f.Request.MakeARequest(method, apiEndpoint, profileConfig.AccessToken, contentType, nil, bytes.NewBuffer(bodyBytes), false)
		if err != nil {
			return "", resp, fmt.Errorf("Error while requesting user access token at %v: %v", apiEndpoint, err)
		}

		// 401 code is general error code from FENCE. the error message is also not clear for the case
		// that the token expired. Temporary solution: get new access token and make another attempt.
		if resp != nil && resp.StatusCode == 401 {
			isExpiredToken = true
		} else {
			return prefixEndPoint, resp, err
		}
	}
	if profileConfig.AccessToken == "" || isExpiredToken {
		err := f.Request.RequestNewAccessToken(prefixEndPoint+commonUtils.FenceAccessTokenEndpoint, profileConfig)
		if err != nil {
			return prefixEndPoint, resp, err
		}
		err = f.Config.UpdateConfigFile(*profileConfig)
		if err != nil {
			return prefixEndPoint, resp, err
		}

		resp, err = f.Request.MakeARequest(method, apiEndpoint, profileConfig.AccessToken, contentType, nil, bytes.NewBuffer(bodyBytes), false)
		if err != nil {
			return prefixEndPoint, resp, err
		}
	}

	return prefixEndPoint, resp, nil
}

func (f *Functions) GetHost(profileConfig *Credential) (*url.URL, error) {
	if profileConfig.APIEndpoint == "" {
		return nil, errors.New("No APIEndpoint found in the configuration file! Please use \"./data-client configure\" to configure your credentials first")
	}
	host, _ := url.Parse(profileConfig.APIEndpoint)
	return host, nil
}

func (f *Functions) DoRequestWithSignedHeader(profileConfig *Credential, endpointPostPrefix string, contentType string, bodyBytes []byte) (JsonMessage, error) {
	/*
	   Do request with signed header. User may have more than one profile and use a profile to make a request
	*/
	var err error
	var msg JsonMessage

	method := "GET"
	if bodyBytes != nil {
		method = "POST"
	}

	_, resp, err := f.GetResponse(profileConfig, endpointPostPrefix, method, contentType, bodyBytes)
	if err != nil {
		return msg, err
	}
	defer resp.Body.Close()

	msg, err = f.ParseFenceURLResponse(resp)
	return msg, err
}

func (f *Functions) CheckPrivileges(profileConfig *Credential) (string, map[string]interface{}, error) {
	/*
	   Return user privileges from specified profile
	*/
	var err error
	var data map[string]interface{}

	host, resp, err := f.GetResponse(profileConfig, commonUtils.FenceUserEndpoint, "GET", "", nil)
	if err != nil {
		return "", nil, errors.New("Error occurred when getting response from remote: " + err.Error())
	}
	defer resp.Body.Close()

	str := ResponseToString(resp)
	err = json.Unmarshal([]byte(str), &data)
	if err != nil {
		return "", nil, errors.New("Error occurred when unmarshalling response: " + err.Error())
	}

	resourceAccess, ok := data["authz"].(map[string]interface{})

	// If the `authz` section (Arborist permissions) is empty or missing, try get `project_access` section (Fence permissions)
	if len(resourceAccess) == 0 || !ok {
		resourceAccess, ok = data["project_access"].(map[string]interface{})
		if !ok {
			return "", nil, errors.New("Not possible to read access privileges of user")
		}
	}

	return host, resourceAccess, err
}

func (f *Functions) DeleteRecord(profileConfig *Credential, guid string) (string, error) {
	var err error
	var msg string

	hasShepherd, err := f.CheckForShepherdAPI(profileConfig)
	if err != nil {
		log.Printf("WARNING: Error while checking for Shepherd API: %v. Falling back to Fence to delete record.\n", err)
	} else if hasShepherd {
		endPointPostfix := commonUtils.ShepherdEndpoint + "/objects/" + guid
		_, resp, err := f.GetResponse(profileConfig, endPointPostfix, "DELETE", "", nil)
		if err != nil {
			return "", err
		}
		defer resp.Body.Close()
		if resp.StatusCode == 204 {
			msg = "Record with GUID " + guid + " has been deleted"
		} else if resp.StatusCode == 500 {
			err = errors.New("Internal server error occurred when deleting " + guid + "; could not delete stored files, or not able to delete INDEXD record")
		}
		return msg, err
	}

	endPointPostfix := commonUtils.FenceDataEndpoint + "/" + guid

	_, resp, err := f.GetResponse(profileConfig, endPointPostfix, "DELETE", "", nil)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode == 204 {
		msg = "Record with GUID " + guid + " has been deleted"
	} else if resp.StatusCode == 500 {
		err = errors.New("Internal server error occurred when deleting " + guid + "; could not delete stored files, or not able to delete INDEXD record")
	}

	return msg, err
}
