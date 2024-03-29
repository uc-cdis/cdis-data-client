// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/uc-cdis/gen3-client/gen3-client/jwt (interfaces: FunctionInterface)

// Package mocks is a generated GoMock package.
package mocks

import (
	gomock "github.com/golang/mock/gomock"
	jwt "github.com/uc-cdis/gen3-client/gen3-client/jwt"
	http "net/http"
	url "net/url"
	reflect "reflect"
)

// MockFunctionInterface is a mock of FunctionInterface interface
type MockFunctionInterface struct {
	ctrl     *gomock.Controller
	recorder *MockFunctionInterfaceMockRecorder
}

// MockFunctionInterfaceMockRecorder is the mock recorder for MockFunctionInterface
type MockFunctionInterfaceMockRecorder struct {
	mock *MockFunctionInterface
}

// NewMockFunctionInterface creates a new mock instance
func NewMockFunctionInterface(ctrl *gomock.Controller) *MockFunctionInterface {
	mock := &MockFunctionInterface{ctrl: ctrl}
	mock.recorder = &MockFunctionInterfaceMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockFunctionInterface) EXPECT() *MockFunctionInterfaceMockRecorder {
	return m.recorder
}

// CheckForShepherdAPI mocks base method
func (m *MockFunctionInterface) CheckForShepherdAPI(arg0 *jwt.Credential) (bool, error) {
	ret := m.ctrl.Call(m, "CheckForShepherdAPI", arg0)
	ret0, _ := ret[0].(bool)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// CheckForShepherdAPI indicates an expected call of CheckForShepherdAPI
func (mr *MockFunctionInterfaceMockRecorder) CheckForShepherdAPI(arg0 interface{}) *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CheckForShepherdAPI", reflect.TypeOf((*MockFunctionInterface)(nil).CheckForShepherdAPI), arg0)
}

// CheckPrivileges mocks base method
func (m *MockFunctionInterface) CheckPrivileges(arg0 *jwt.Credential) (string, map[string]interface{}, error) {
	ret := m.ctrl.Call(m, "CheckPrivileges", arg0)
	ret0, _ := ret[0].(string)
	ret1, _ := ret[1].(map[string]interface{})
	ret2, _ := ret[2].(error)
	return ret0, ret1, ret2
}

// CheckPrivileges indicates an expected call of CheckPrivileges
func (mr *MockFunctionInterfaceMockRecorder) CheckPrivileges(arg0 interface{}) *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CheckPrivileges", reflect.TypeOf((*MockFunctionInterface)(nil).CheckPrivileges), arg0)
}

// DoRequestWithSignedHeader mocks base method
func (m *MockFunctionInterface) DoRequestWithSignedHeader(arg0 *jwt.Credential, arg1, arg2 string, arg3 []byte) (jwt.JsonMessage, error) {
	ret := m.ctrl.Call(m, "DoRequestWithSignedHeader", arg0, arg1, arg2, arg3)
	ret0, _ := ret[0].(jwt.JsonMessage)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// DoRequestWithSignedHeader indicates an expected call of DoRequestWithSignedHeader
func (mr *MockFunctionInterfaceMockRecorder) DoRequestWithSignedHeader(arg0, arg1, arg2, arg3 interface{}) *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DoRequestWithSignedHeader", reflect.TypeOf((*MockFunctionInterface)(nil).DoRequestWithSignedHeader), arg0, arg1, arg2, arg3)
}

// GetHost mocks base method
func (m *MockFunctionInterface) GetHost(arg0 *jwt.Credential) (*url.URL, error) {
	ret := m.ctrl.Call(m, "GetHost", arg0)
	ret0, _ := ret[0].(*url.URL)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetHost indicates an expected call of GetHost
func (mr *MockFunctionInterfaceMockRecorder) GetHost(arg0 interface{}) *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetHost", reflect.TypeOf((*MockFunctionInterface)(nil).GetHost), arg0)
}

// GetResponse mocks base method
func (m *MockFunctionInterface) GetResponse(arg0 *jwt.Credential, arg1, arg2, arg3 string, arg4 []byte) (string, *http.Response, error) {
	ret := m.ctrl.Call(m, "GetResponse", arg0, arg1, arg2, arg3, arg4)
	ret0, _ := ret[0].(string)
	ret1, _ := ret[1].(*http.Response)
	ret2, _ := ret[2].(error)
	return ret0, ret1, ret2
}

// GetResponse indicates an expected call of GetResponse
func (mr *MockFunctionInterfaceMockRecorder) GetResponse(arg0, arg1, arg2, arg3, arg4 interface{}) *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetResponse", reflect.TypeOf((*MockFunctionInterface)(nil).GetResponse), arg0, arg1, arg2, arg3, arg4)
}

// ParseFenceURLResponse mocks base method
func (m *MockFunctionInterface) ParseFenceURLResponse(arg0 *http.Response) (jwt.JsonMessage, error) {
	ret := m.ctrl.Call(m, "ParseFenceURLResponse", arg0)
	ret0, _ := ret[0].(jwt.JsonMessage)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ParseFenceURLResponse indicates an expected call of ParseFenceURLResponse
func (mr *MockFunctionInterfaceMockRecorder) ParseFenceURLResponse(arg0 interface{}) *gomock.Call {
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ParseFenceURLResponse", reflect.TypeOf((*MockFunctionInterface)(nil).ParseFenceURLResponse), arg0)
}
