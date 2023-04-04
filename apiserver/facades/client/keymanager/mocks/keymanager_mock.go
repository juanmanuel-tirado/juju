// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/juju/juju/apiserver/facades/client/keymanager (interfaces: Model,BlockChecker)

// Package mocks is a generated GoMock package.
package mocks

import (
	reflect "reflect"

	gomock "github.com/golang/mock/gomock"
	config "github.com/juju/juju/environs/config"
	state "github.com/juju/juju/state"
	names "github.com/juju/names/v4"
)

// MockModel is a mock of Model interface.
type MockModel struct {
	ctrl     *gomock.Controller
	recorder *MockModelMockRecorder
}

// MockModelMockRecorder is the mock recorder for MockModel.
type MockModelMockRecorder struct {
	mock *MockModel
}

// NewMockModel creates a new mock instance.
func NewMockModel(ctrl *gomock.Controller) *MockModel {
	mock := &MockModel{ctrl: ctrl}
	mock.recorder = &MockModelMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockModel) EXPECT() *MockModelMockRecorder {
	return m.recorder
}

// ModelConfig mocks base method.
func (m *MockModel) ModelConfig() (*config.Config, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ModelConfig")
	ret0, _ := ret[0].(*config.Config)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ModelConfig indicates an expected call of ModelConfig.
func (mr *MockModelMockRecorder) ModelConfig() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ModelConfig", reflect.TypeOf((*MockModel)(nil).ModelConfig))
}

// ModelTag mocks base method.
func (m *MockModel) ModelTag() names.ModelTag {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ModelTag")
	ret0, _ := ret[0].(names.ModelTag)
	return ret0
}

// ModelTag indicates an expected call of ModelTag.
func (mr *MockModelMockRecorder) ModelTag() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ModelTag", reflect.TypeOf((*MockModel)(nil).ModelTag))
}

// UpdateModelConfig mocks base method.
func (m *MockModel) UpdateModelConfig(arg0 map[string]interface{}, arg1 []string, arg2 ...state.ValidateConfigFunc) error {
	m.ctrl.T.Helper()
	varargs := []interface{}{arg0, arg1}
	for _, a := range arg2 {
		varargs = append(varargs, a)
	}
	ret := m.ctrl.Call(m, "UpdateModelConfig", varargs...)
	ret0, _ := ret[0].(error)
	return ret0
}

// UpdateModelConfig indicates an expected call of UpdateModelConfig.
func (mr *MockModelMockRecorder) UpdateModelConfig(arg0, arg1 interface{}, arg2 ...interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	varargs := append([]interface{}{arg0, arg1}, arg2...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UpdateModelConfig", reflect.TypeOf((*MockModel)(nil).UpdateModelConfig), varargs...)
}

// MockBlockChecker is a mock of BlockChecker interface.
type MockBlockChecker struct {
	ctrl     *gomock.Controller
	recorder *MockBlockCheckerMockRecorder
}

// MockBlockCheckerMockRecorder is the mock recorder for MockBlockChecker.
type MockBlockCheckerMockRecorder struct {
	mock *MockBlockChecker
}

// NewMockBlockChecker creates a new mock instance.
func NewMockBlockChecker(ctrl *gomock.Controller) *MockBlockChecker {
	mock := &MockBlockChecker{ctrl: ctrl}
	mock.recorder = &MockBlockCheckerMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockBlockChecker) EXPECT() *MockBlockCheckerMockRecorder {
	return m.recorder
}

// ChangeAllowed mocks base method.
func (m *MockBlockChecker) ChangeAllowed() error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ChangeAllowed")
	ret0, _ := ret[0].(error)
	return ret0
}

// ChangeAllowed indicates an expected call of ChangeAllowed.
func (mr *MockBlockCheckerMockRecorder) ChangeAllowed() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ChangeAllowed", reflect.TypeOf((*MockBlockChecker)(nil).ChangeAllowed))
}

// RemoveAllowed mocks base method.
func (m *MockBlockChecker) RemoveAllowed() error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "RemoveAllowed")
	ret0, _ := ret[0].(error)
	return ret0
}

// RemoveAllowed indicates an expected call of RemoveAllowed.
func (mr *MockBlockCheckerMockRecorder) RemoveAllowed() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "RemoveAllowed", reflect.TypeOf((*MockBlockChecker)(nil).RemoveAllowed))
}
