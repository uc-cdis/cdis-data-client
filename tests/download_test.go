package tests

import (
	"errors"
	"net/http"
	"net/url"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/uc-cdis/cdis-data-client/cmd"
	"github.com/uc-cdis/cdis-data-client/jwt"
	"github.com/uc-cdis/cdis-data-client/mocks"
)

func TestRequestDownloadPanic(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("The code did not panic")
		}
	}()

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockFunction := mocks.NewMockFunctionInterface(mockCtrl)
	testFunction := &cmd.Download{Function: mockFunction}

	cred := jwt.Credential{KeyId: "fake_keyid", APIKey: "fake_api_key", AccessKey: "fake_access_key", APIEndpoint: "http://fence.com/download"}
	mockedResp := &http.Response{}

	mockFunction.EXPECT().SignedRequest("GET", "http://test.com/user/data/download/", nil, cred.AccessKey).Return(mockedResp, errors.New("dummy code")).Times(1)

	u, _ := url.Parse("http://test.com/index.html")

	testFunction.RequestDownload(cred, u, "json")

}

func TestSignRequestCalled(t *testing.T) {

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockFunction := mocks.NewMockFunctionInterface(mockCtrl)
	mockUtils := mocks.NewMockUtilInterface(mockCtrl)
	mockRequest := mocks.NewMockRequestInterface(mockCtrl)
	testFunction := &cmd.Download{Function: mockFunction, Request: mockRequest, Utils: mockUtils}

	cred := jwt.Credential{KeyId: "fake_keyid", APIKey: "fake_api_key", AccessKey: "fake_access_key", APIEndpoint: "http://fence.com/download"}

	mockedResp := &http.Response{}
	//mockedResp.StatusCode = 401

	mockFunction.EXPECT().SignedRequest("GET", "http://test.com/user/data/download/", nil, cred.AccessKey).Return(mockedResp, nil).Times(1)
	mockUtils.EXPECT().ResponseToString(mockedResp).Return("http://www.google.com")
	u, _ := url.Parse("http://test.com/index.html")

	res := testFunction.RequestDownload(cred, u, "json")
	if res == nil {
		t.Fail()
	}

}

func TestRequestNewTokenCalled(t *testing.T) {

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockFunction := mocks.NewMockFunctionInterface(mockCtrl)
	mockUtils := mocks.NewMockUtilInterface(mockCtrl)
	mockRequest := mocks.NewMockRequestInterface(mockCtrl)
	testFunction := &cmd.Download{Function: mockFunction, Request: mockRequest, Utils: mockUtils}

	cred := jwt.Credential{KeyId: "fake_keyid", APIKey: "fake_api_key", AccessKey: "fake_access_key", APIEndpoint: "http://fence.com/download"}

	mockedResp := &http.Response{}
	mockedClient := &http.Client{}
	mockedResp.StatusCode = 401

	mockFunction.EXPECT().SignedRequest("GET", "http://test.com/user/data/download/", nil, cred.AccessKey).Return(mockedResp, nil).Times(2)
	mockUtils.EXPECT().ResponseToString(mockedResp).Return("http://www.google.com").Times(2)

	mockRequest.EXPECT().RequestNewAccessKey(mockedClient, cred.APIEndpoint+"/credentials/cdis/access_token", &cred).Times(1)
	u, _ := url.Parse("http://test.com/index.html")

	res := testFunction.RequestDownload(cred, u, "json")
	if res == nil {
		t.Fail()
	}

}
