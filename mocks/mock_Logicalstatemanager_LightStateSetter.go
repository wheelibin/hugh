// Code generated by mockery v2.36.1. DO NOT EDIT.

package mocks

import (
	time "time"

	mock "github.com/stretchr/testify/mock"
)

// MockLogicalstatemanagerLightStateSetter is an autogenerated mock type for the lightStateSetter type
type MockLogicalstatemanagerLightStateSetter struct {
	mock.Mock
}

type MockLogicalstatemanagerLightStateSetter_Expecter struct {
	mock *mock.Mock
}

func (_m *MockLogicalstatemanagerLightStateSetter) EXPECT() *MockLogicalstatemanagerLightStateSetter_Expecter {
	return &MockLogicalstatemanagerLightStateSetter_Expecter{mock: &_m.Mock}
}

// SetLightStateToTarget provides a mock function with given fields: lsID, currentTime
func (_m *MockLogicalstatemanagerLightStateSetter) SetLightStateToTarget(lsID string, currentTime time.Time) error {
	ret := _m.Called(lsID, currentTime)

	var r0 error
	if rf, ok := ret.Get(0).(func(string, time.Time) error); ok {
		r0 = rf(lsID, currentTime)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// MockLogicalstatemanagerLightStateSetter_SetLightStateToTarget_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'SetLightStateToTarget'
type MockLogicalstatemanagerLightStateSetter_SetLightStateToTarget_Call struct {
	*mock.Call
}

// SetLightStateToTarget is a helper method to define mock.On call
//   - lsID string
//   - currentTime time.Time
func (_e *MockLogicalstatemanagerLightStateSetter_Expecter) SetLightStateToTarget(lsID interface{}, currentTime interface{}) *MockLogicalstatemanagerLightStateSetter_SetLightStateToTarget_Call {
	return &MockLogicalstatemanagerLightStateSetter_SetLightStateToTarget_Call{Call: _e.mock.On("SetLightStateToTarget", lsID, currentTime)}
}

func (_c *MockLogicalstatemanagerLightStateSetter_SetLightStateToTarget_Call) Run(run func(lsID string, currentTime time.Time)) *MockLogicalstatemanagerLightStateSetter_SetLightStateToTarget_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run(args[0].(string), args[1].(time.Time))
	})
	return _c
}

func (_c *MockLogicalstatemanagerLightStateSetter_SetLightStateToTarget_Call) Return(_a0 error) *MockLogicalstatemanagerLightStateSetter_SetLightStateToTarget_Call {
	_c.Call.Return(_a0)
	return _c
}

func (_c *MockLogicalstatemanagerLightStateSetter_SetLightStateToTarget_Call) RunAndReturn(run func(string, time.Time) error) *MockLogicalstatemanagerLightStateSetter_SetLightStateToTarget_Call {
	_c.Call.Return(run)
	return _c
}

// NewMockLogicalstatemanagerLightStateSetter creates a new instance of MockLogicalstatemanagerLightStateSetter. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewMockLogicalstatemanagerLightStateSetter(t interface {
	mock.TestingT
	Cleanup(func())
}) *MockLogicalstatemanagerLightStateSetter {
	mock := &MockLogicalstatemanagerLightStateSetter{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
