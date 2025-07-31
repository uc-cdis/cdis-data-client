package tests

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/calypr/data-client/data-client/jwt"
	"github.com/calypr/data-client/data-client/mocks"
	"github.com/golang/mock/gomock"
)

func TestDoRequestWithSignedHeaderNoProfile(t *testing.T) {

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockConfig := mocks.NewMockConfigureInterface(mockCtrl)
	testFunction := &jwt.Functions{Config: mockConfig}

	profileConfig := jwt.Credential{KeyId: "", APIKey: "", AccessToken: "", APIEndpoint: ""}

	_, err := testFunction.DoRequestWithSignedHeader(&profileConfig, "/user/data/download/test_uuid", "", nil)

	if err == nil {
		t.Fail()
	}
}

func TestDoRequestWithSignedHeaderGoodToken(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockConfig := mocks.NewMockConfigureInterface(mockCtrl)
	mockRequest := mocks.NewMockRequestInterface(mockCtrl)
	testFunction := &jwt.Functions{Config: mockConfig, Request: mockRequest}

	profileConfig := jwt.Credential{Profile: "test", KeyId: "", APIKey: "fake_api_key", AccessToken: "non_expired_token", APIEndpoint: "http://www.test.com", UseShepherd: "false", MinShepherdVersion: ""}
	mockedResp := &http.Response{
		Body:       io.NopCloser(bytes.NewBufferString("{\"url\": \"http://www.test.com/user/data/download/test_uuid\"}")),
		StatusCode: 200,
	}

	mockRequest.EXPECT().MakeARequest("GET", "http://www.test.com/user/data/download/test_uuid", "non_expired_token", "", gomock.Any(), gomock.Any(), false).Return(mockedResp, nil).Times(1)

	_, err := testFunction.DoRequestWithSignedHeader(&profileConfig, "/user/data/download/test_uuid", "", nil)

	if err != nil {
		t.Fail()
	}
}

func TestDoRequestWithSignedHeaderCreateNewToken(t *testing.T) {

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockConfig := mocks.NewMockConfigureInterface(mockCtrl)
	mockRequest := mocks.NewMockRequestInterface(mockCtrl)
	testFunction := &jwt.Functions{Config: mockConfig, Request: mockRequest}

	profileConfig := jwt.Credential{KeyId: "", APIKey: "fake_api_key", AccessToken: "", APIEndpoint: "http://www.test.com"}
	mockedResp := &http.Response{
		Body:       io.NopCloser(bytes.NewBufferString("{\"url\": \"www.test.com/user/data/download/\"}")),
		StatusCode: 200,
	}

	mockConfig.EXPECT().UpdateConfigFile(profileConfig).Times(1)
	mockRequest.EXPECT().RequestNewAccessToken("http://www.test.com/user/credentials/api/access_token", &profileConfig).Return(nil).Times(1)
	mockRequest.EXPECT().MakeARequest(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), false).Return(mockedResp, nil).Times(1)

	_, err := testFunction.DoRequestWithSignedHeader(&profileConfig, "/user/data/download/test_uuid", "", nil)

	if err != nil {
		t.Fail()
	}
}

func TestDoRequestWithSignedHeaderRefreshToken(t *testing.T) {

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockConfig := mocks.NewMockConfigureInterface(mockCtrl)
	mockRequest := mocks.NewMockRequestInterface(mockCtrl)
	testFunction := &jwt.Functions{Config: mockConfig, Request: mockRequest}

	profileConfig := jwt.Credential{KeyId: "", APIKey: "fake_api_key", AccessToken: "expired_token", APIEndpoint: "http://www.test.com"}
	mockedResp := &http.Response{
		Body:       io.NopCloser(bytes.NewBufferString("{\"url\": \"www.test.com/user/data/download/\"}")),
		StatusCode: 401,
	}

	mockConfig.EXPECT().UpdateConfigFile(profileConfig).Times(1)
	mockRequest.EXPECT().RequestNewAccessToken("http://www.test.com/user/credentials/api/access_token", &profileConfig).Return(nil).Times(1)
	mockRequest.EXPECT().MakeARequest(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), false).Return(mockedResp, nil).Times(2)

	_, err := testFunction.DoRequestWithSignedHeader(&profileConfig, "/user/data/download/test_uuid", "", nil)

	if err != nil && !strings.Contains(err.Error(), "401") {
		t.Fail()
	}

}

func TestCheckPrivilegesNoProfile(t *testing.T) {

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockConfig := mocks.NewMockConfigureInterface(mockCtrl)
	testFunction := &jwt.Functions{Config: mockConfig}

	profileConfig := jwt.Credential{KeyId: "", APIKey: "", AccessToken: "", APIEndpoint: ""}

	_, _, err := testFunction.CheckPrivileges(&profileConfig)

	if err == nil {
		t.Errorf("Expected an error on missing credentials in configuration, but not received")
	}
}

func TestCheckPrivilegesNoAccess(t *testing.T) {

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockConfig := mocks.NewMockConfigureInterface(mockCtrl)
	mockRequest := mocks.NewMockRequestInterface(mockCtrl)
	testFunction := &jwt.Functions{Config: mockConfig, Request: mockRequest}

	profileConfig := jwt.Credential{KeyId: "", APIKey: "fake_api_key", AccessToken: "non_expired_token", APIEndpoint: "http://www.test.com"}
	mockedResp := &http.Response{
		Body:       io.NopCloser(bytes.NewBufferString("{\"project_access\": {}}")),
		StatusCode: 200,
	}

	mockRequest.EXPECT().MakeARequest("GET", "http://www.test.com/user/user", "non_expired_token", "", gomock.Any(), gomock.Any(), false).Return(mockedResp, nil).Times(1)

	_, receivedAccess, err := testFunction.CheckPrivileges(&profileConfig)

	expectedAccess := make(map[string]interface{})

	if err != nil {
		t.Errorf("Expected no errors, received an error \"%v\"", err)
	} else if !reflect.DeepEqual(receivedAccess, expectedAccess) {
		t.Errorf("Expected no user access, received %v", receivedAccess)
	}
}

func TestCheckPrivilegesGrantedAccess(t *testing.T) {

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockConfig := mocks.NewMockConfigureInterface(mockCtrl)
	mockRequest := mocks.NewMockRequestInterface(mockCtrl)
	testFunction := &jwt.Functions{Config: mockConfig, Request: mockRequest}

	profileConfig := jwt.Credential{KeyId: "", APIKey: "fake_api_key", AccessToken: "non_expired_token", APIEndpoint: "http://www.test.com"}

	grantedAccessJSON := `{
		"project_access":
			{
				"test_project": ["read", "create","read-storage","update","delete"]
			}
		}`

	mockedResp := &http.Response{
		Body:       io.NopCloser(bytes.NewBufferString(grantedAccessJSON)),
		StatusCode: 200,
	}

	mockRequest.EXPECT().MakeARequest("GET", "http://www.test.com/user/user", "non_expired_token", "", gomock.Any(), gomock.Any(), false).Return(mockedResp, nil).Times(1)

	_, expectedAccess, err := testFunction.CheckPrivileges(&profileConfig)

	receivedAccess := make(map[string]interface{})
	receivedAccess["test_project"] = []interface{}{
		"read",
		"create",
		"read-storage",
		"update",
		"delete"}

	if err != nil {
		t.Errorf("Expected no errors, received an error \"%v\"", err)
	} else if !reflect.DeepEqual(expectedAccess, receivedAccess) {
		t.Errorf(`Expected user access and received user access are not the same.
        Expected: %v
        Received: %v`, expectedAccess, receivedAccess)
	}
}

// If both `authz` and `project_access` section exists, `authz` takes precedence
func TestCheckPrivilegesGrantedAccessAuthz(t *testing.T) {

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockConfig := mocks.NewMockConfigureInterface(mockCtrl)
	mockRequest := mocks.NewMockRequestInterface(mockCtrl)
	testFunction := &jwt.Functions{Config: mockConfig, Request: mockRequest}

	profileConfig := jwt.Credential{KeyId: "", APIKey: "fake_api_key", AccessToken: "non_expired_token", APIEndpoint: "http://www.test.com"}

	grantedAccessJSON := `{
		"authz": {
			"test_project":[
				{"method":"create", "service":"*"},
				{"method":"delete", "service":"*"},
				{"method":"read", "service":"*"},
				{"method":"read-storage", "service":"*"},
				{"method":"update", "service":"*"},
				{"method":"upload", "service":"*"}
			]
		},
		"project_access": {
			"test_project": ["read", "create","read-storage","update","delete"]
		}
	}`

	mockedResp := &http.Response{
		Body:       io.NopCloser(bytes.NewBufferString(grantedAccessJSON)),
		StatusCode: 200,
	}

	mockRequest.EXPECT().MakeARequest("GET", "http://www.test.com/user/user", "non_expired_token", "", gomock.Any(), gomock.Any(), false).Return(mockedResp, nil).Times(1)

	_, expectedAccess, err := testFunction.CheckPrivileges(&profileConfig)

	receivedAccess := make(map[string]interface{})
	receivedAccess["test_project"] = []map[string]interface{}{
		{"method": "create", "service": "*"},
		{"method": "delete", "service": "*"},
		{"method": "read", "service": "*"},
		{"method": "read-storage", "service": "*"},
		{"method": "update", "service": "*"},
		{"method": "upload", "service": "*"},
	}

	if err != nil {
		t.Errorf("Expected no errors, received an error \"%v\"", err)
		// don't use DeepEqual since expectedAccess is []interface {} and receivedAccess is []map[string]interface {}, just check for contents
	} else if fmt.Sprint(expectedAccess) != fmt.Sprint(receivedAccess) {
		t.Errorf(`Expected user access and received user access are not the same.
        Expected: %v
        Received: %v`, expectedAccess, receivedAccess)
	}
}
