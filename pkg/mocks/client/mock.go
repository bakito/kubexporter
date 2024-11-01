// Code generated by MockGen. DO NOT EDIT.
// Source: k8s.io/client-go/dynamic (interfaces: Interface)
//
// Generated by this command:
//
//	mockgen -destination pkg/mocks/client/mock.go k8s.io/client-go/dynamic Interface
//

// Package mock_dynamic is a generated GoMock package.
package mock_dynamic

import (
	reflect "reflect"

	gomock "go.uber.org/mock/gomock"
	schema "k8s.io/apimachinery/pkg/runtime/schema"
	dynamic "k8s.io/client-go/dynamic"
)

// MockInterface is a mock of Interface interface.
type MockInterface struct {
	ctrl     *gomock.Controller
	recorder *MockInterfaceMockRecorder
	isgomock struct{}
}

// MockInterfaceMockRecorder is the mock recorder for MockInterface.
type MockInterfaceMockRecorder struct {
	mock *MockInterface
}

// NewMockInterface creates a new mock instance.
func NewMockInterface(ctrl *gomock.Controller) *MockInterface {
	mock := &MockInterface{ctrl: ctrl}
	mock.recorder = &MockInterfaceMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockInterface) EXPECT() *MockInterfaceMockRecorder {
	return m.recorder
}

// Resource mocks base method.
func (m *MockInterface) Resource(resource schema.GroupVersionResource) dynamic.NamespaceableResourceInterface {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Resource", resource)
	ret0, _ := ret[0].(dynamic.NamespaceableResourceInterface)
	return ret0
}

// Resource indicates an expected call of Resource.
func (mr *MockInterfaceMockRecorder) Resource(resource any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Resource", reflect.TypeOf((*MockInterface)(nil).Resource), resource)
}
