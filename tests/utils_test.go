package tests

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/uc-cdis/gen3-client/gen3-client/commonUtils"
	g3cmd "github.com/uc-cdis/gen3-client/gen3-client/g3cmd"
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
	testProfileConfig := &jwt.Credential{
		Profile: "test-profile",
	}
	testFilename := "test-file"
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	// Mock the request that checks if Shepherd is deployed.
	mockGen3Interface := mocks.NewMockGen3Interface(mockCtrl)
	mockGen3Interface.
		EXPECT().
		CheckForShepherdAPI(gomock.AssignableToTypeOf(testProfileConfig)).
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
		GetResponse(gomock.AssignableToTypeOf(testProfileConfig), commonUtils.ShepherdEndpoint+"/objects/"+testGUID+"/download", "GET", "", nil).
		Return("", &mockDownloadURLResponse, nil)

	// Mock the request for the file at mockDownloadURL.
	mockFileResponse := http.Response{
		StatusCode: 200,
		Body:       ioutil.NopCloser(strings.NewReader("It work")),
	}
	mockGen3Interface.
		EXPECT().
		MakeARequest(http.MethodGet, mockDownloadURL, "", "", map[string]string{}, nil, true).
		Return(&mockFileResponse, nil)
	// ----------

	mockFDRObj := commonUtils.FileDownloadResponseObject{
		Filename: testFilename,
		GUID:     testGUID,
		Range:    0,
	}
	err := g3cmd.GetDownloadResponse(mockGen3Interface, &mockFDRObj, "")
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
	testProfileConfig := &jwt.Credential{
		Profile: "test-profile",
	}
	testFilename := "test-file"
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	// Mock the request that checks if Shepherd is deployed.
	mockGen3Interface := mocks.NewMockGen3Interface(mockCtrl)
	mockGen3Interface.
		EXPECT().
		CheckForShepherdAPI(gomock.AssignableToTypeOf(testProfileConfig)).
		Return(false, nil)

	// Mock the request to Fence for the download URL of this file.
	mockDownloadURL := "https://example.com/example.pfb"
	mockDownloadURLResponse := jwt.JsonMessage{
		URL: mockDownloadURL,
	}
	mockGen3Interface.
		EXPECT().
		DoRequestWithSignedHeader(gomock.AssignableToTypeOf(testProfileConfig), commonUtils.FenceDataDownloadEndpoint+"/"+testGUID, "", nil).
		Return(mockDownloadURLResponse, nil)

	// Mock the request for the file at mockDownloadURL.
	mockFileResponse := http.Response{
		StatusCode: 200,
		Body:       ioutil.NopCloser(strings.NewReader("It work")),
	}
	mockGen3Interface.
		EXPECT().
		MakeARequest(http.MethodGet, mockDownloadURL, "", "", map[string]string{}, nil, true).
		Return(&mockFileResponse, nil)
	// ----------

	mockFDRObj := commonUtils.FileDownloadResponseObject{
		Filename: testFilename,
		GUID:     testGUID,
		Range:    0,
	}
	err := g3cmd.GetDownloadResponse(mockGen3Interface, &mockFDRObj, "")
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

// If Shepherd is not deployed, expect GeneratePresignedURL to hit fence's data upload
// endpoint and return the presigned URL and guid.
func TestGeneratePresignedURL_noShepherd(t *testing.T) {
	// -- SETUP --
	testProfileConfig := &jwt.Credential{
		Profile: "test-profile",
	}
	testFilename := "test-file"
	testBucketname := "test-bucket"
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	// Mock the request that checks if Shepherd is deployed.
	mockGen3Interface := mocks.NewMockGen3Interface(mockCtrl)
	mockGen3Interface.
		EXPECT().
		CheckForShepherdAPI(gomock.AssignableToTypeOf(testProfileConfig)).
		Return(false, nil)

	// Mock the request to Fence's data upload endpoint to create a presigned url for this file name.
	expectedReqBody := []byte(fmt.Sprintf(`{"file_name":"%v", "bucket":"%v"}`, testFilename, testBucketname))
	mockPresignedURL := "https://example.com/example.pfb"
	mockGUID := "000000-0000000-0000000-000000"
	mockUploadURLResponse := jwt.JsonMessage{
		URL:  mockPresignedURL,
		GUID: mockGUID,
	}
	mockGen3Interface.
		EXPECT().
		DoRequestWithSignedHeader(gomock.AssignableToTypeOf(testProfileConfig), commonUtils.FenceDataUploadEndpoint, "application/json", expectedReqBody).
		Return(mockUploadURLResponse, nil)
	// ----------

	url, guid, err := g3cmd.GeneratePresignedURL(mockGen3Interface, testFilename, commonUtils.FileMetadata{}, testBucketname)
	if err != nil {
		t.Error(err)
	}
	if url != mockPresignedURL {
		t.Errorf("Wanted the presignedURL to be set to %v, got %v", mockPresignedURL, url)
	}
	if guid != mockGUID {
		t.Errorf("Wanted generated GUID to be %v, got %v", mockGUID, guid)
	}
}

// If Shepherd is deployed, expect GeneratePresignedURL to hit Shepherd's data upload
// endpoint with the file name and file metadata. GeneratePresignedURL should then
// return the guid and file name that it gets from the endpoint.
func TestGeneratePresignedURL_withShepherd(t *testing.T) {
	// -- SETUP --
	testProfileConfig := &jwt.Credential{
		Profile: "test-profile",
	}
	testFilename := "test-file"
	testBucketname := "test-bucket"
	testMetadata := commonUtils.FileMetadata{
		Aliases:  []string{"test-alias-1", "test-alias-2"},
		Authz:    []string{"authz-resource-1", "authz-resource-2"},
		Metadata: map[string]interface{}{"arbitrary": "metadata"},
	}
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	// Mock the request that checks if Shepherd is deployed.
	mockGen3Interface := mocks.NewMockGen3Interface(mockCtrl)
	mockGen3Interface.
		EXPECT().
		CheckForShepherdAPI(gomock.AssignableToTypeOf(testProfileConfig)).
		Return(true, nil)

	// Mock the request to Fence's data upload endpoint to create a presigned url for this file name.
	expectedReq := g3cmd.ShepherdInitRequestObject{
		Filename: testFilename,
		Authz: struct {
			Version       string   `json:"version"`
			ResourcePaths []string `json:"resource_paths"`
		}{
			"0",
			testMetadata.Authz,
		},
		Aliases:  testMetadata.Aliases,
		Metadata: testMetadata.Metadata,
	}
	expectedReqBody, err := json.Marshal(expectedReq)
	if err != nil {
		t.Error(err)
	}
	mockPresignedURL := "https://example.com/example.pfb"
	mockGUID := "000000-0000000-0000000-000000"
	presignedURLBody := fmt.Sprintf(`{
		"guid": "%v",
		"upload_url": "%v"
	}`, mockGUID, mockPresignedURL)
	mockUploadURLResponse := http.Response{
		StatusCode: 201,
		Body:       ioutil.NopCloser(strings.NewReader(presignedURLBody)),
	}
	mockGen3Interface.
		EXPECT().
		GetResponse(gomock.AssignableToTypeOf(testProfileConfig), commonUtils.ShepherdEndpoint+"/objects", "POST", "", expectedReqBody).
		Return("", &mockUploadURLResponse, nil)
	// ----------

	url, guid, err := g3cmd.GeneratePresignedURL(mockGen3Interface, testFilename, testMetadata, testBucketname)
	if err != nil {
		t.Error(err)
	}
	if url != mockPresignedURL {
		t.Errorf("Wanted the presignedURL to be set to %v, got %v", mockPresignedURL, url)
	}
	if guid != mockGUID {
		t.Errorf("Wanted generated GUID to be %v, got %v", mockGUID, guid)
	}
}
