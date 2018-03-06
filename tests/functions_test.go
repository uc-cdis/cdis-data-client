package tests

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/uc-cdis/cdis-data-client/jwt"
	"github.com/uc-cdis/cdis-data-client/mocks"
)

func Requesting(*http.Response) *http.Response {
	return &http.Response{}
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

	testFunction.DoRequestWithSignedHeader(Requesting, "default", "not_json", "/user/data/download/test_uuid")

}

func TestDoRequestWithSignedHeaderGoodToken(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockConfig := mocks.NewMockConfigureInterface(mockCtrl)
	mockRequest := mocks.NewMockRequestInterface(mockCtrl)
	testFunction := &jwt.Functions{Config: mockConfig, Request: mockRequest}

	cred := jwt.Credential{KeyId: "", APIKey: "fake_api_key", AccessKey: "non_exprired_token", APIEndpoint: "http://www.test.com"}
	mockedResp := &http.Response{
		Body:       ioutil.NopCloser(bytes.NewBufferString("{'url': 'http://www.test.com/user/data/download/test_uuid'}")),
		StatusCode: 200,
	}

	mockConfig.EXPECT().ParseConfig("default").Return(cred).Times(1)
	mockRequest.EXPECT().GetPresignedURL(gomock.Any(), "/user/data/download/test_uuid", "non_exprired_token").Return(mockedResp).Times(1)

	res := testFunction.DoRequestWithSignedHeader(Requesting, "default", "", "/user/data/download/test_uuid")

	if res == nil {
		t.Fail()
	}
}
func TestDoRequestWithSignedHeaderCreateNewToken(t *testing.T) {

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockConfig := mocks.NewMockConfigureInterface(mockCtrl)
	mockRequest := mocks.NewMockRequestInterface(mockCtrl)
	testFunction := &jwt.Functions{Config: mockConfig, Request: mockRequest}

	cred := jwt.Credential{KeyId: "", APIKey: "fake_api_key", AccessKey: "", APIEndpoint: "http://www.test.com"}
	mockedResp := &http.Response{
		Body:       ioutil.NopCloser(bytes.NewBufferString("{'url': 'www.test.com/user/data/download/'}")),
		StatusCode: 200,
	}

	mockConfig.EXPECT().ParseConfig("default").Return(cred).Times(1)
	mockConfig.EXPECT().ReadFile(gomock.Any(), gomock.Any()).Times(1)
	mockConfig.EXPECT().UpdateConfigFile(cred, gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(1)

	mockRequest.EXPECT().RequestNewAccessKey("http://www.test.com/user/credentials/cdis/access_token", &cred).Times(1)
	mockRequest.EXPECT().GetPresignedURL(gomock.Any(), gomock.Any(), gomock.Any()).Return(mockedResp).Times(1)

	res := testFunction.DoRequestWithSignedHeader(Requesting, "default", "", "/user/data/download/test_uuid")

	if res == nil {
		t.Fail()
	}

}

func TestDoRequestWithSignedHeaderRefreshToken(t *testing.T) {

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockConfig := mocks.NewMockConfigureInterface(mockCtrl)
	mockRequest := mocks.NewMockRequestInterface(mockCtrl)
	testFunction := &jwt.Functions{Config: mockConfig, Request: mockRequest}

	cred := jwt.Credential{KeyId: "", APIKey: "fake_api_key", AccessKey: "expired_token", APIEndpoint: "http://www.test.com"}
	mockedResp := &http.Response{
		Body:       ioutil.NopCloser(bytes.NewBufferString("{'url': 'www.test.com/user/data/download/'}")),
		StatusCode: 401,
	}

	mockConfig.EXPECT().ParseConfig("default").Return(cred).Times(1)
	mockConfig.EXPECT().ReadFile(gomock.Any(), gomock.Any()).Times(1)
	mockConfig.EXPECT().UpdateConfigFile(cred, gomock.Any(), "http://www.test.com", gomock.Any(), "default").Times(1)

	mockRequest.EXPECT().RequestNewAccessKey("http://www.test.com/user/credentials/cdis/access_token", &cred).Times(1)
	mockRequest.EXPECT().GetPresignedURL(gomock.Any(), gomock.Any(), gomock.Any()).Return(mockedResp).Times(2)

	res := testFunction.DoRequestWithSignedHeader(Requesting, "default", "", "/user/data/download/test_uuid")

	if res == nil {
		t.Fail()
	}

}
