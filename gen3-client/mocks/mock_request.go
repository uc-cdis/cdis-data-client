// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/uc-cdis/gen3-client/gen3-client/jwt (interfaces: RequestInterface)

// Package mocks is a generated GoMock package.
package mocks

import (
	bytes "bytes"
	gomock "github.com/golang/mock/gomock"
	jwt "github.com/uc-cdis/gen3-client/gen3-client/jwt"
	http "net/http"
	reflect "reflect"
)

// MockRequestInterface is a mock of RequestInterface interface
type MockRequestInterface struct {
	ctrl     *gomock.Controller
	recorder *MockRequestInterfaceMockRecorder
}

// MockRequestInterfaceMockRecorder is the mock recorder for MockRequestInterface
type MockRequestInterfaceMockRecorder struct {
	mock *MockRequestInterface
}

// NewMockRequestInterface creates a new mock instance
func NewMockRequestInterface(ctrl *gomock.Controller) *MockRequestInterface {
	mock := &MockRequestInterface{ctrl: ctrl}
	mock.recorder = &MockRequestInterfaceMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockRequestInterface) EXPECT() *MockRequestInterfaceMockRecorder {
	return m.recorder
}

// MakeARequest mocks base method
func (m *MockRequestInterface) MakeARequest(arg0, arg1, arg2, arg3 string, arg4 map[string]string, arg5 *bytes.Buffer) (*http.Response, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "MakeARequest", arg0, arg1, arg2, arg3, arg4, arg5)
	ret0, _ := ret[0].(*http.Response)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// MakeARequest indicates an expected call of MakeARequest
func (mr *MockRequestInterfaceMockRecorder) MakeARequest(arg0, arg1, arg2, arg3, arg4, arg5 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "MakeARequest", reflect.TypeOf((*MockRequestInterface)(nil).MakeARequest), arg0, arg1, arg2, arg3, arg4, arg5)
}

// RequestNewAccessKey mocks base method
func (m *MockRequestInterface) RequestNewAccessKey(arg0 string, arg1 *jwt.Credential) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "RequestNewAccessKey", arg0, arg1)
	ret0, _ := ret[0].(error)
	return ret0
}

// RequestNewAccessKey indicates an expected call of RequestNewAccessKey
func (mr *MockRequestInterfaceMockRecorder) RequestNewAccessKey(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "RequestNewAccessKey", reflect.TypeOf((*MockRequestInterface)(nil).RequestNewAccessKey), arg0, arg1)
}
