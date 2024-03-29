// Code generated by MockGen. DO NOT EDIT.
// Source: collector/prometheus.go

// Package mocks is a generated GoMock package.
package mocks

import (
	context "context"
	reflect "reflect"
	time "time"

	gomock "github.com/golang/mock/gomock"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	model "github.com/prometheus/common/model"
)

// MockPrometheusConnection is a mock of PrometheusConnection interface.
type MockPrometheusConnection struct {
	ctrl     *gomock.Controller
	recorder *MockPrometheusConnectionMockRecorder
}

// MockPrometheusConnectionMockRecorder is the mock recorder for MockPrometheusConnection.
type MockPrometheusConnectionMockRecorder struct {
	mock *MockPrometheusConnection
}

// NewMockPrometheusConnection creates a new mock instance.
func NewMockPrometheusConnection(ctrl *gomock.Controller) *MockPrometheusConnection {
	mock := &MockPrometheusConnection{ctrl: ctrl}
	mock.recorder = &MockPrometheusConnectionMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockPrometheusConnection) EXPECT() *MockPrometheusConnectionMockRecorder {
	return m.recorder
}

// Query mocks base method.
func (m *MockPrometheusConnection) Query(ctx context.Context, query string, ts time.Time, opts ...v1.Option) (model.Value, v1.Warnings, error) {
	m.ctrl.T.Helper()
	varargs := []interface{}{ctx, query, ts}
	for _, a := range opts {
		varargs = append(varargs, a)
	}
	ret := m.ctrl.Call(m, "Query", varargs...)
	ret0, _ := ret[0].(model.Value)
	ret1, _ := ret[1].(v1.Warnings)
	ret2, _ := ret[2].(error)
	return ret0, ret1, ret2
}

// Query indicates an expected call of Query.
func (mr *MockPrometheusConnectionMockRecorder) Query(ctx, query, ts interface{}, opts ...interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	varargs := append([]interface{}{ctx, query, ts}, opts...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Query", reflect.TypeOf((*MockPrometheusConnection)(nil).Query), varargs...)
}

// QueryRange mocks base method.
func (m *MockPrometheusConnection) QueryRange(ctx context.Context, query string, r v1.Range, opts ...v1.Option) (model.Value, v1.Warnings, error) {
	m.ctrl.T.Helper()
	varargs := []interface{}{ctx, query, r}
	for _, a := range opts {
		varargs = append(varargs, a)
	}
	ret := m.ctrl.Call(m, "QueryRange", varargs...)
	ret0, _ := ret[0].(model.Value)
	ret1, _ := ret[1].(v1.Warnings)
	ret2, _ := ret[2].(error)
	return ret0, ret1, ret2
}

// QueryRange indicates an expected call of QueryRange.
func (mr *MockPrometheusConnectionMockRecorder) QueryRange(ctx, query, r interface{}, opts ...interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	varargs := append([]interface{}{ctx, query, r}, opts...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "QueryRange", reflect.TypeOf((*MockPrometheusConnection)(nil).QueryRange), varargs...)
}
