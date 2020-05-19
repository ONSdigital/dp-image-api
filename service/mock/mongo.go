// Code generated by moq; DO NOT EDIT.
// github.com/matryer/moq

package mock

import (
	"context"
	"github.com/ONSdigital/dp-healthcheck/healthcheck"
	"github.com/ONSdigital/dp-image-api/service"
	"sync"
)

var (
	lockIMongoMockChecker sync.RWMutex
	lockIMongoMockClose   sync.RWMutex
)

// Ensure, that IMongoMock does implement service.IMongo.
// If this is not the case, regenerate this file with moq.
var _ service.IMongo = &IMongoMock{}

// IMongoMock is a mock implementation of service.IMongo.
//
//     func TestSomethingThatUsesIMongo(t *testing.T) {
//
//         // make and configure a mocked service.IMongo
//         mockedIMongo := &IMongoMock{
//             CheckerFunc: func(ctx context.Context, state *healthcheck.CheckState) error {
// 	               panic("mock out the Checker method")
//             },
//             CloseFunc: func(ctx context.Context) error {
// 	               panic("mock out the Close method")
//             },
//         }
//
//         // use mockedIMongo in code that requires service.IMongo
//         // and then make assertions.
//
//     }
type IMongoMock struct {
	// CheckerFunc mocks the Checker method.
	CheckerFunc func(ctx context.Context, state *healthcheck.CheckState) error

	// CloseFunc mocks the Close method.
	CloseFunc func(ctx context.Context) error

	// calls tracks calls to the methods.
	calls struct {
		// Checker holds details about calls to the Checker method.
		Checker []struct {
			// Ctx is the ctx argument value.
			Ctx context.Context
			// State is the state argument value.
			State *healthcheck.CheckState
		}
		// Close holds details about calls to the Close method.
		Close []struct {
			// Ctx is the ctx argument value.
			Ctx context.Context
		}
	}
}

// Checker calls CheckerFunc.
func (mock *IMongoMock) Checker(ctx context.Context, state *healthcheck.CheckState) error {
	if mock.CheckerFunc == nil {
		panic("IMongoMock.CheckerFunc: method is nil but IMongo.Checker was just called")
	}
	callInfo := struct {
		Ctx   context.Context
		State *healthcheck.CheckState
	}{
		Ctx:   ctx,
		State: state,
	}
	lockIMongoMockChecker.Lock()
	mock.calls.Checker = append(mock.calls.Checker, callInfo)
	lockIMongoMockChecker.Unlock()
	return mock.CheckerFunc(ctx, state)
}

// CheckerCalls gets all the calls that were made to Checker.
// Check the length with:
//     len(mockedIMongo.CheckerCalls())
func (mock *IMongoMock) CheckerCalls() []struct {
	Ctx   context.Context
	State *healthcheck.CheckState
} {
	var calls []struct {
		Ctx   context.Context
		State *healthcheck.CheckState
	}
	lockIMongoMockChecker.RLock()
	calls = mock.calls.Checker
	lockIMongoMockChecker.RUnlock()
	return calls
}

// Close calls CloseFunc.
func (mock *IMongoMock) Close(ctx context.Context) error {
	if mock.CloseFunc == nil {
		panic("IMongoMock.CloseFunc: method is nil but IMongo.Close was just called")
	}
	callInfo := struct {
		Ctx context.Context
	}{
		Ctx: ctx,
	}
	lockIMongoMockClose.Lock()
	mock.calls.Close = append(mock.calls.Close, callInfo)
	lockIMongoMockClose.Unlock()
	return mock.CloseFunc(ctx)
}

// CloseCalls gets all the calls that were made to Close.
// Check the length with:
//     len(mockedIMongo.CloseCalls())
func (mock *IMongoMock) CloseCalls() []struct {
	Ctx context.Context
} {
	var calls []struct {
		Ctx context.Context
	}
	lockIMongoMockClose.RLock()
	calls = mock.calls.Close
	lockIMongoMockClose.RUnlock()
	return calls
}
