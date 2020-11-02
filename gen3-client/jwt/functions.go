package jwt

//mockgen -destination=mocks/mock_functions.go -package=mocks jwt FunctionInterface

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"strings"

	"github.com/hashicorp/go-version"
	homedir "github.com/mitchellh/go-homedir"
	"github.com/uc-cdis/gen3-client/gen3-client/commonUtils"
)

type Functions struct {
	Request RequestInterface
	Config  ConfigureInterface
}

type FunctionInterface interface {
	CheckForShepherdAPI(string) (bool, error)
	GetResponse(string, string, string, string, string, []byte) (string, *http.Response, error)
	DoRequestWithSignedHeader(string, string, string, string, []byte) (JsonMessage, error)
	ParseFenceURLResponse(*http.Response) (JsonMessage, error)
}

type Request struct {
}

type RequestInterface interface {
	MakeARequest(string, string, string, string, map[string]string, *bytes.Buffer) (*http.Response, error)
	RequestNewAccessKey(string, *Credential) error
}

func (r *Request) MakeARequest(method string, apiEndpoint string, accessKey string, contentType string, headers map[string]string, body *bytes.Buffer) (*http.Response, error) {
	/*
		Make http request with header and body
	*/
	if headers == nil {
		headers = make(map[string]string)
	}
	if accessKey != "" {
		headers["Authorization"] = "Bearer " + accessKey
	}
	if contentType != "" {
		headers["Content-Type"] = contentType
	}
	client := &http.Client{Timeout: commonUtils.DefaultTimeout}
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
	resp, err := r.MakeARequest("POST", apiEndpoint, "", "application/json", nil, body)
	var m AccessTokenStruct
	// parse resp error codes first for profile configuration verification
	if resp != nil && resp.StatusCode != 200 {
		return errors.New("Error occurred in RequestNewAccessKey with error code " + strconv.Itoa(resp.StatusCode) + ", check FENCE log for more details.")
	}
	if err != nil {
		return errors.New("Error occurred in RequestNewAccessKey: " + err.Error())
	}

	str := ResponseToString(resp)
	err = DecodeJsonFromString(str, &m)
	if err != nil {
		return errors.New("Error occurred in RequestNewAccessKey: " + err.Error())
	}

	if m.AccessToken == "" {
		return errors.New("Could not get new access key from response string: " + str)
	}
	cred.AccessKey = m.AccessToken
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
			return msg, errors.New("404 Not found error has occurred! The requested url \"" + resp.Request.URL.String() + "\" cannot be found")
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

func (f *Functions) CheckForShepherdAPI(profile string) (bool, error) {
	// Check if Shepherd is enabled
	cred := f.Config.ParseConfig(profile)
	if cred.UseShepherd == "false" {
		return false, nil
	}
	if cred.UseShepherd != "true" && commonUtils.DefaultUseShepherd == false {
		return false, nil
	}
	// If Shepherd is enabled, make sure that the commons has a compatible version of Shepherd deployed.
	// Compare the version returned from the Shepherd version endpoint with the minimum acceptable Shepherd version.
	var minShepherdVersion string
	if cred.MinShepherdVersion == "" {
		minShepherdVersion = commonUtils.DefaultMinShepherdVersion
	} else {
		minShepherdVersion = cred.MinShepherdVersion
	}

	_, res, err := f.GetResponse(profile, "", commonUtils.ShepherdVersionEndpoint, "GET", "", nil)
	if err != nil {
		return false, errors.New("Error occurred during generating HTTP request: " + err.Error())
	}
	if res.StatusCode != 200 {
		return false, nil
	}
	bodyBytes, err := ioutil.ReadAll(res.Body)
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
	return false, fmt.Errorf("Shepherd is enabled, but %v does not have correct Shepherd version. (Need Shepherd version >=%v, got %v)", cred.APIEndpoint, minVer, ver)
}

func (f *Functions) GetResponse(profile string, configFileType string, endpointPostPrefix string, method string, contentType string, bodyBytes []byte) (string, *http.Response, error) {

	var resp *http.Response
	var err error

	cred := f.Config.ParseConfig(profile)
	if cred.APIKey == "" && cred.AccessKey == "" && cred.APIEndpoint == "" {
		return "", resp, errors.New("No credentials found in the configuration file! Please use \"./gen3-client configure\" to configure your credentials first")
	}
	host, _ := url.Parse(cred.APIEndpoint)
	prefixEndPoint := host.Scheme + "://" + host.Host
	apiEndpoint := host.Scheme + "://" + host.Host + endpointPostPrefix
	isExpiredToken := false
	if cred.AccessKey != "" {
		resp, err = f.Request.MakeARequest(method, apiEndpoint, cred.AccessKey, contentType, nil, bytes.NewBuffer(bodyBytes))
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
	if cred.AccessKey == "" || isExpiredToken {
		err := f.Request.RequestNewAccessKey(prefixEndPoint+commonUtils.FenceAccessTokenEndpoint, &cred)
		if err != nil {
			return prefixEndPoint, resp, err
		}
		homeDir, err := homedir.Dir()
		if err != nil {
			log.Fatalln("Error occurred when getting home directory: " + err.Error())
		}
		configPath := path.Join(homeDir + commonUtils.PathSeparator + ".gen3" + commonUtils.PathSeparator + "config")
		content := f.Config.ReadFile(configPath, configFileType)
		f.Config.UpdateConfigFile(cred, []byte(content), cred.APIEndpoint, cred.UseShepherd, cred.MinShepherdVersion, configPath, profile)

		resp, err = f.Request.MakeARequest(method, apiEndpoint, cred.AccessKey, contentType, nil, bytes.NewBuffer(bodyBytes))
		if err != nil {
			return prefixEndPoint, resp, err
		}
	}

	return prefixEndPoint, resp, nil
}

func (f *Functions) GetHost(profile string, configFileType string) (*url.URL, error) {
	cred := f.Config.ParseConfig(profile)
	if cred.APIEndpoint == "" {
		return nil, errors.New("No APIEndpoint found in the configuration file! Please use \"./gen3-client configure\" to configure your credentials first")
	}
	host, _ := url.Parse(cred.APIEndpoint)
	return host, nil
}

func (f *Functions) DoRequestWithSignedHeader(profile string, configFileType string, endpointPostPrefix string, contentType string, bodyBytes []byte) (JsonMessage, error) {
	/*
	   Do request with signed header. User may have more than one profile and use a profile to make a request
	*/
	var err error
	var msg JsonMessage

	method := "GET"
	if bodyBytes != nil {
		method = "POST"
	}

	_, resp, err := f.GetResponse(profile, configFileType, endpointPostPrefix, method, contentType, bodyBytes)
	if err != nil {
		return msg, err
	}

	msg, err = f.ParseFenceURLResponse(resp)
	return msg, err
}

func (f *Functions) CheckProfileConfig(apiKey string, apiEndpoint string) error {
	return nil
}

func (f *Functions) CheckPrivileges(profile string, configFileType string) (string, map[string]interface{}, error) {
	/*
	   Return user privileges from specified profile
	*/
	var err error
	var data map[string]interface{}

	host, resp, err := f.GetResponse(profile, configFileType, commonUtils.FenceUserEndpoint, "GET", "", nil)
	if err != nil {
		return "", nil, errors.New("Error occurred when getting response from remote: " + err.Error())
	}

	str := ResponseToString(resp)

	err = json.Unmarshal([]byte(str), &data)
	if err != nil {
		return "", nil, errors.New("Error occurred when unmarshalling response: " + err.Error())
	}

	projectAccess, ok := data["project_access"].(map[string]interface{})
	if !ok {
		return "", nil, errors.New("Not possible to read user access privileges")
	}

	return host, projectAccess, err
}

func (f *Functions) DeleteRecord(profile string, configFileType string, guid string) (string, error) {
	var err error
	var msg string

	hasShepherd, err := f.CheckForShepherdAPI(profile)
	if err != nil {
		log.Printf("WARNING: Error while checking for Shepherd API: %v. Falling back to Fence to delete record.\n", err)
	} else if hasShepherd {
		endPointPostfix := commonUtils.ShepherdEndpoint + "/objects/" + guid
		_, resp, err := f.GetResponse(profile, configFileType, endPointPostfix, "DELETE", "", nil)
		if err != nil {
			return "", err
		}
		if resp.StatusCode == 204 {
			msg = "Record with GUID " + guid + " has been deleted"
		} else if resp.StatusCode == 500 {
			err = errors.New("Internal server error occurred when deleting " + guid + "; could not delete stored files, or not able to delete INDEXD record")
		}
		return msg, err
	}

	endPointPostfix := commonUtils.FenceDataEndpoint + "/" + guid

	_, resp, err := f.GetResponse(profile, configFileType, endPointPostfix, "DELETE", "", nil)

	if resp.StatusCode == 204 {
		msg = "Record with GUID " + guid + " has been deleted"
	} else if resp.StatusCode == 500 {
		err = errors.New("Internal server error occurred when deleting " + guid + "; could not delete stored files, or not able to delete INDEXD record")
	}

	return msg, err
}
