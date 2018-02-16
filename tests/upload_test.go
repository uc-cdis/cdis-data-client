package tests

import (
	"bytes"
	"errors"
	"io/ioutil"
	"net/http"
	"net/url"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/uc-cdis/cdis-data-client/cmd"
	"github.com/uc-cdis/cdis-data-client/jwt"
	"github.com/uc-cdis/cdis-data-client/mocks"
)

func TestRequestUploadPanic(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("The code did not panic")
		}
	}()

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockFunction := mocks.NewMockFunctionInterface(mockCtrl)
	testFunction := &cmd.Upload{Function: mockFunction}

	cred := jwt.Credential{KeyId: "fake_keyid", APIKey: "fake_api_key", AccessKey: "fake_access_key", APIEndpoint: "http://fence.com/download"}
	mockedResp := &http.Response{}

	mockFunction.EXPECT().SignedRequest("GET", "http://test.com/user/data/upload/", nil, cred.AccessKey).Return(mockedResp, errors.New("dummy code")).Times(1)

	u, _ := url.Parse("http://test.com/index.html")

	testFunction.GetPreSignedURL(cred, u, "json")

}

func TestSignUpCalled(t *testing.T) {

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockFunction := mocks.NewMockFunctionInterface(mockCtrl)
	mockRequest := mocks.NewMockRequestInterface(mockCtrl)
	testFunction := &cmd.Upload{Function: mockFunction, Request: mockRequest}

	cred := jwt.Credential{KeyId: "fake_keyid", APIKey: "fake_api_key", AccessKey: "fake_access_key", APIEndpoint: "http://fence.com/download"}

	mockedResp := &http.Response{
		Body: ioutil.NopCloser(bytes.NewBufferString("{'url': 'www.test.com'}")),
	}

	// io.ReadCloser.Read(mockedResp.Body, []byte("{'url': 'google.com'}"))
	// //mockedResp.StatusCode = 401

	mockFunction.EXPECT().SignedRequest("GET", "http://test.com/user/data/upload/", nil, cred.AccessKey).Return(mockedResp, nil).Times(1)
	u, _ := url.Parse("http://test.com/index.html")
	testFunction.GetPreSignedURL(cred, u, "json")
}

func TestUploadRequestNewTokenCalled(t *testing.T) {

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockFunction := mocks.NewMockFunctionInterface(mockCtrl)
	mockRequest := mocks.NewMockRequestInterface(mockCtrl)
	testFunction := &cmd.Upload{Function: mockFunction, Request: mockRequest}

	cred := jwt.Credential{KeyId: "fake_keyid", APIKey: "fake_api_key", AccessKey: "expired_token", APIEndpoint: "http://fence.com/download"}

	mockedResp := &http.Response{
		Body:       ioutil.NopCloser(bytes.NewBufferString("{'url': 'www.test.com'}")),
		StatusCode: 401,
	}

	mockFunction.EXPECT().SignedRequest("GET", "http://test.com/user/data/upload/", nil, cred.AccessKey).Return(mockedResp, nil).Times(2)

	mockRequest.EXPECT().RequestNewAccessKey(gomock.Any(), cred.APIEndpoint+"/credentials/cdis/access_token", &cred).Times(1)
	u, _ := url.Parse("http://test.com/index.html")

	testFunction.GetPreSignedURL(cred, u, "json")

}
