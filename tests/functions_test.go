package tests

import (
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/uc-cdis/cdis-data-client/jwt"
	"github.com/uc-cdis/cdis-data-client/mocks"
)

func assertPanic(t *testing.T, f func()) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("The code did not panic")
		}
	}()
	f()
}
func TestDoRequestWithSignedHeaderNoProfile(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("The code did not panic")
		}
	}()

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockConfig := mocks.NewMockConfigureInterface(mockCtrl)
	testFunction := &jwt.Functions{Config: mockConfig}

	cred := jwt.Credential{KeyId: "", APIKey: "", AccessKey: "", APIEndpoint: ""}

	mockConfig.EXPECT().ParseConfig(gomock.Any()).Return(cred).Times(1)

	function := jwt.Functions{}

	testFunction.DoRequestWithSignedHeader(function.Requesting, "default", "notjson")

}

func TestDoRequestWithSignedHeaderGoodToken(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockConfig := mocks.NewMockConfigureInterface(mockCtrl)
	mockRequest := mocks.NewMockRequestInterface(mockCtrl)
	testFunction := &jwt.Functions{Config: mockConfig, Request: mockRequest}

	cred := jwt.Credential{KeyId: "", APIKey: "fake_api_key", AccessKey: "non_exprired_token", APIEndpoint: "test.com"}
	mockConfig.EXPECT().ParseConfig("default").Return(cred).Times(1)

	function := new(jwt.Functions)
	res, _ := testFunction.DoRequestWithSignedHeader(function.Requesting, "default", "")
	if res == nil {
		t.Fail()
	}
}
func TestDoRequestWithSignedHeaderCreateNewTokenCalled(t *testing.T) {

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockConfig := mocks.NewMockConfigureInterface(mockCtrl)
	mockRequest := mocks.NewMockRequestInterface(mockCtrl)
	testFunction := &jwt.Functions{Config: mockConfig, Request: mockRequest}

	cred := jwt.Credential{KeyId: "", APIKey: "fake_api_key", AccessKey: "", APIEndpoint: "test.com"}

	mockConfig.EXPECT().ParseConfig("default").Return(cred).Times(1)
	mockConfig.EXPECT().ReadFile(gomock.Any(), gomock.Any()).Times(1)
	mockConfig.EXPECT().UpdateConfigFile(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(1)

	mockRequest.EXPECT().RequestNewAccessKey(cred.APIEndpoint+"/credentials/cdis/access_token", &cred).Times(1)

	function := new(jwt.Functions)

	res, _ := testFunction.DoRequestWithSignedHeader(function.Requesting, "default", "")
	if res == nil {
		t.Fail()
	}

}
