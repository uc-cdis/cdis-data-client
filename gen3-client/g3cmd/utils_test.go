package g3cmd

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/uc-cdis/gen3-client/gen3-client/commonUtils"
	"github.com/uc-cdis/gen3-client/gen3-client/jwt"
	"github.com/uc-cdis/gen3-client/gen3-client/mocks"
)

// Expect GetDownloadResponse to:
// 1. get the file download URL from Shepherd if it's deployed
// 2. add the file download URL to the FileDownloadResponseObject
// 3. GET the file download URL, and add the response to the FileDownloadResponseObject
func TestGetDownloadResponse_withShepherd(t *testing.T) {
	// -- SETUP --
	testGUID := "000000-0000000-0000000-000000"
	testProfile := "test-profile"
	testFilename := "test-file"
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	// Mock the request that checks if Shepherd is deployed.
	mockGen3Interface := mocks.NewMockGen3Interface(mockCtrl)
	mockGen3Interface.
		EXPECT().
		CheckForShepherdAPI(testProfile).
		Return(true, nil)

	// Mock the request to Shepherd for the download URL of this file.
	mockDownloadURL := "https://example.com/example.pfb"
	downloadURLBody := fmt.Sprintf(`{
		"url": "%v"
	}`, mockDownloadURL)
	mockDownloadURLResponse := http.Response{
		StatusCode: 200,
		Body:       ioutil.NopCloser(strings.NewReader(downloadURLBody)),
	}
	mockGen3Interface.
		EXPECT().
		GetResponse(testProfile, "", commonUtils.ShepherdEndpoint+"/objects/"+testGUID+"/download", "GET", "", nil).
		Return("", &mockDownloadURLResponse, nil)

	// Mock the request for the file at mockDownloadURL.
	mockFileResponse := http.Response{
		StatusCode: 200,
		Body:       ioutil.NopCloser(strings.NewReader("It work")),
	}
	mockGen3Interface.
		EXPECT().
		MakeARequest("GET", mockDownloadURL, "", "", map[string]string{}, nil).
		Return(&mockFileResponse, nil)
	// ----------

	mockFDRObj := commonUtils.FileDownloadResponseObject{
		Filename: testFilename,
		GUID:     testGUID,
		Range:    0,
	}
	err := GetDownloadResponse(mockGen3Interface, testProfile, &mockFDRObj, "")
	if err != nil {
		t.Error(err)
	}
	if mockFDRObj.URL != mockDownloadURL {
		t.Errorf("Wanted the DownloadPath to be set to %v, got %v", mockDownloadURL, mockFDRObj.DownloadPath)
	}
	if mockFDRObj.Response != &mockFileResponse {
		t.Errorf("Wanted download response to be %v, got %v", mockFileResponse, mockFDRObj.Response)
	}
}

// Expect GetDownloadResponse to:
// 1. get the file download URL from Fence if Shepherd is not deployed
// 2. add the file download URL to the FileDownloadResponseObject
// 3. GET the file download URL, and add the response to the FileDownloadResponseObject
func TestGetDownloadResponse_noShepherd(t *testing.T) {
	// -- SETUP --
	testGUID := "000000-0000000-0000000-000000"
	testProfile := "test-profile"
	testFilename := "test-file"
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	// Mock the request that checks if Shepherd is deployed.
	mockGen3Interface := mocks.NewMockGen3Interface(mockCtrl)
	mockGen3Interface.
		EXPECT().
		CheckForShepherdAPI(testProfile).
		Return(false, nil)

	// Mock the request to Fence for the download URL of this file.
	mockDownloadURL := "https://example.com/example.pfb"
	mockDownloadURLResponse := jwt.JsonMessage{
		URL: mockDownloadURL,
	}
	mockGen3Interface.
		EXPECT().
		DoRequestWithSignedHeader(testProfile, "", commonUtils.FenceDataDownloadEndpoint+"/"+testGUID, "", nil).
		Return(mockDownloadURLResponse, nil)

	// Mock the request for the file at mockDownloadURL.
	mockFileResponse := http.Response{
		StatusCode: 200,
		Body:       ioutil.NopCloser(strings.NewReader("It work")),
	}
	mockGen3Interface.
		EXPECT().
		MakeARequest("GET", mockDownloadURL, "", "", map[string]string{}, nil).
		Return(&mockFileResponse, nil)
	// ----------

	mockFDRObj := commonUtils.FileDownloadResponseObject{
		Filename: testFilename,
		GUID:     testGUID,
		Range:    0,
	}
	err := GetDownloadResponse(mockGen3Interface, testProfile, &mockFDRObj, "")
	if err != nil {
		t.Error(err)
	}
	if mockFDRObj.URL != mockDownloadURL {
		t.Errorf("Wanted the DownloadPath to be set to %v, got %v", mockDownloadURL, mockFDRObj.DownloadPath)
	}
	if mockFDRObj.Response != &mockFileResponse {
		t.Errorf("Wanted download response to be %v, got %v", mockFileResponse, mockFDRObj.Response)
	}
}
