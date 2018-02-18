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

func TestDownloadGetPreSignedURLPanic(t *testing.T) {
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

	mockedResp := &http.Response{
		Body:       ioutil.NopCloser(bytes.NewBufferString("{'url': 'www.test.com'}")),
		StatusCode: 401,
	}

	mockFunction.EXPECT().SignedRequest("GET", "http://test.com/user/data/download/", nil, cred.AccessKey).Return(mockedResp, errors.New("dummy code")).Times(1)

	u, _ := url.Parse("http://test.com/index.html")

	testFunction.GetDownloadPreSignedURL(cred, u, "json")

}

func TestDownloadGetPreSignedURLReturnPresignedURL(t *testing.T) {
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockFunction := mocks.NewMockFunctionInterface(mockCtrl)
	testFunction := &cmd.Download{Function: mockFunction}

	cred := jwt.Credential{KeyId: "fake_keyid", APIKey: "fake_api_key", AccessKey: "fake_access_key", APIEndpoint: "http://fence.com/download"}
	mockedResp := &http.Response{
		Body:       ioutil.NopCloser(bytes.NewBufferString("{'url': 'www.test.com'}")),
		StatusCode: 200,
	}
	mockFunction.EXPECT().SignedRequest("GET", "http://fence.com/user/data/download/", nil, cred.AccessKey).Return(mockedResp, nil).Times(1)

	u, _ := url.Parse(cred.APIEndpoint)

	testFunction.GetDownloadPreSignedURL(cred, u, "json")
}
