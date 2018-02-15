package tests

import (
	"os/user"
	"path"
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
func TestNoProfile(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("The code did not panic")
		}
	}()

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockUtils := mocks.NewMockUtilInterface(mockCtrl)
	mockConfigure := mocks.NewMockConfigureInterface(nil)
	mockRequest := mocks.NewMockRequestInterface(nil)
	testFunction := &jwt.Functions{Config: mockConfigure, Request: mockRequest, Utils: mockUtils}

	cred := jwt.Credential{KeyId: "", APIKey: "", AccessKey: "", APIEndpoint: ""}

	mockUtils.EXPECT().ParseConfig("default").Return(cred).AnyTimes()

	function := jwt.Functions{}

	testFunction.DoRequestWithSignedHeader(function.Requesting, "default")

}

func TestReturnNil(t *testing.T) {

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockUtils := mocks.NewMockUtilInterface(mockCtrl)
	mockConfigure := mocks.NewMockConfigureInterface(nil)
	mockRequest := mocks.NewMockRequestInterface(nil)
	testFunction := &jwt.Functions{Config: mockConfigure, Request: mockRequest, Utils: mockUtils}

	cred := jwt.Credential{KeyId: "", APIKey: "", AccessKey: "fake_access_key", APIEndpoint: ""}

	mockUtils.EXPECT().ParseConfig("default").Return(cred).AnyTimes()

	function := jwt.Functions{}

	res := testFunction.DoRequestWithSignedHeader(function.Requesting, "default")
	if res != nil {
		t.Fail()
	}

}

func TestReturnNotNil(t *testing.T) {

	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	mockUtils := mocks.NewMockUtilInterface(mockCtrl)
	mockConfigure := mocks.NewMockConfigureInterface(mockCtrl)
	mockRequest := mocks.NewMockRequestInterface(mockCtrl)
	testFunction := &jwt.Functions{Config: mockConfigure, Request: mockRequest, Utils: new(jwt.Utils)}

	cred := jwt.Credential{KeyId: "", APIKey: "fake_api_key", AccessKey: "", APIEndpoint: ""}

	mockUtils.EXPECT().ParseConfig("default").Return(cred).AnyTimes()

	usr, _ := user.Current()
	homeDir := usr.HomeDir
	configPath := path.Join(homeDir + "/.cdis/config")
	mockConfigure.EXPECT().ReadFile(configPath, "").Return("").AnyTimes()

	function := new(jwt.Functions)

	res := testFunction.DoRequestWithSignedHeader(function.Requesting, "default")
	if res != nil {
		t.Fail()
	}

}
