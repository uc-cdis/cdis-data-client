package tests

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/calypr/data-client/data-client/commonUtils"
	g3cmd "github.com/calypr/data-client/data-client/g3cmd"
	"github.com/calypr/data-client/data-client/jwt"
	"github.com/calypr/data-client/data-client/mocks"
	"github.com/golang/mock/gomock"
)

// If Shepherd is deployed, attempt to get the filename from the Shepherd API.
func Test_askGen3ForFileInfo_withShepherd(t *testing.T) {
	// -- SETUP --
	testGUID := "000000-0000000-0000000-000000"
	testProfileConfig := &jwt.Credential{
		Profile: "test-profile",
	}
	testFileName := "test-file"
	testFileSize := int64(120)
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	// Expect AskGen3ForFileInfo to call shepherd looking for testGUID: respond with a valid file.
	testBody := `{
	"record": {
		"file_name": "test-file",
		"size": 120,
		"did": "000000-0000000-0000000-000000"
	},
	"metadata": {
		"_file_type": "PFB",
		"_resource_paths": ["/open"],
		"_uploader_id": 42,
		"_bucket": "s3://gen3-bucket"
	}
}`
	testResponse := http.Response{
		StatusCode: 200,
		Body:       io.NopCloser(strings.NewReader(testBody)),
	}
	mockGen3Interface := mocks.NewMockGen3Interface(mockCtrl)
	mockGen3Interface.
		EXPECT().
		CheckForShepherdAPI(gomock.AssignableToTypeOf(testProfileConfig)).
		Return(true, nil)
	mockGen3Interface.
		EXPECT().
		GetResponse(gomock.AssignableToTypeOf(testProfileConfig), commonUtils.ShepherdEndpoint+"/objects/"+testGUID, "GET", "", nil).
		Return("", &testResponse, nil)
	// ----------

	// Expect AskGen3ForFileInfo to return the correct filename and filesize from shepherd.
	fileName, fileSize := g3cmd.AskGen3ForFileInfo(mockGen3Interface, testGUID, "", "", "original", true, &[]g3cmd.RenamedOrSkippedFileInfo{})
	if fileName != testFileName {
		t.Errorf("Wanted filename %v, got %v", testFileName, fileName)
	}
	if fileSize != testFileSize {
		t.Errorf("Wanted filesize %v, got %v", testFileSize, fileSize)
	}
}

// If there's an error while getting the filename from Shepherd, add the guid
// to *renamedFiles, which tracks which files have errored.
func Test_askGen3ForFileInfo_withShepherd_shepherdError(t *testing.T) {
	// -- SETUP --
	testGUID := "000000-0000000-0000000-000000"
	testProfileConfig := &jwt.Credential{
		Profile: "test-profile",
	}
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	// Expect AskGen3ForFileInfo to call indexd looking for testGUID:
	// Respond with an error.
	mockGen3Interface := mocks.NewMockGen3Interface(mockCtrl)
	mockGen3Interface.
		EXPECT().
		CheckForShepherdAPI(gomock.AssignableToTypeOf(testProfileConfig)).
		Return(true, nil)
	mockGen3Interface.
		EXPECT().
		GetResponse(gomock.AssignableToTypeOf(testProfileConfig), commonUtils.ShepherdEndpoint+"/objects/"+testGUID, "GET", "", nil).
		Return("", nil, fmt.Errorf("Error getting metadata from Shepherd"))
	// ----------

	// Expect AskGen3ForFileInfo to add this file's GUID to the renamedOrSkippedFiles array.
	skipped := []g3cmd.RenamedOrSkippedFileInfo{}
	fileName, _ := g3cmd.AskGen3ForFileInfo(mockGen3Interface, testGUID, "", "", "original", true, &skipped)
	expected := g3cmd.RenamedOrSkippedFileInfo{GUID: testGUID, OldFilename: "N/A", NewFilename: testGUID}
	if skipped[0] != expected {
		t.Errorf("Wanted skipped files list to contain %v, got %v", expected, skipped)
	}
	// Expect the returned filename to be the file's GUID.
	if fileName != testGUID {
		t.Errorf("Wanted filename %v, got %v", testGUID, fileName)
	}
}

// If Shepherd is not deployed, attempt to get the filename from indexd.
func Test_askGen3ForFileInfo_noShepherd(t *testing.T) {
	// -- SETUP --
	testGUID := "000000-0000000-0000000-000000"
	testProfileConfig := &jwt.Credential{
		Profile: "test-profile",
	}
	testFileName := "test-file"
	testFileSize := int64(120)
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	// Expect AskGen3ForFileInfo to call indexd looking for testGUID: respond with a valid file.
	mockGen3Interface := mocks.NewMockGen3Interface(mockCtrl)
	mockGen3Interface.
		EXPECT().
		CheckForShepherdAPI(gomock.AssignableToTypeOf(testProfileConfig)).
		Return(false, nil)
	mockGen3Interface.
		EXPECT().
		DoRequestWithSignedHeader(gomock.AssignableToTypeOf(testProfileConfig), commonUtils.IndexdIndexEndpoint+"/"+testGUID, "", nil).
		Return(jwt.JsonMessage{FileName: testFileName, Size: testFileSize}, nil)
	// ----------

	// Expect AskGen3ForFileInfo to return the correct filename and filesize from indexd.
	fileName, fileSize := g3cmd.AskGen3ForFileInfo(mockGen3Interface, testGUID, "", "", "original", true, &[]g3cmd.RenamedOrSkippedFileInfo{})
	if fileName != testFileName {
		t.Errorf("Wanted filename %v, got %v", testFileName, fileName)
	}
	if fileSize != testFileSize {
		t.Errorf("Wanted filesize %v, got %v", testFileSize, fileSize)
	}
}

// If there's an error while getting the filename from indexd, add the guid
// to *renamedFiles, which tracks which files have errored.
func Test_askGen3ForFileInfo_noShepherd_indexdError(t *testing.T) {
	// -- SETUP --
	testGUID := "000000-0000000-0000000-000000"
	testProfileConfig := &jwt.Credential{
		Profile: "test-profile",
	}
	mockCtrl := gomock.NewController(t)
	defer mockCtrl.Finish()

	// Expect AskGen3ForFileInfo to call indexd looking for testGUID:
	// Respond with an error.
	mockGen3Interface := mocks.NewMockGen3Interface(mockCtrl)
	mockGen3Interface.
		EXPECT().
		CheckForShepherdAPI(gomock.AssignableToTypeOf(testProfileConfig)).
		Return(false, nil)
	mockGen3Interface.
		EXPECT().
		DoRequestWithSignedHeader(gomock.AssignableToTypeOf(testProfileConfig), commonUtils.IndexdIndexEndpoint+"/"+testGUID, "", nil).
		Return(jwt.JsonMessage{}, fmt.Errorf("Error downloading file from Indexd"))
	// ----------

	// Expect AskGen3ForFileInfo to add this file's GUID to the renamedOrSkippedFiles array.
	skipped := []g3cmd.RenamedOrSkippedFileInfo{}
	fileName, _ := g3cmd.AskGen3ForFileInfo(mockGen3Interface, testGUID, "", "", "original", true, &skipped)
	expected := g3cmd.RenamedOrSkippedFileInfo{GUID: testGUID, OldFilename: "N/A", NewFilename: testGUID}
	if skipped[0] != expected {
		t.Errorf("Wanted skipped files list to contain %v, got %v", expected, skipped)
	}
	// Expect the returned filename to be the file's GUID.
	if fileName != testGUID {
		t.Errorf("Wanted filename %v, got %v", testGUID, fileName)
	}
}
