// Code generated by moq; DO NOT EDIT.
// github.com/matryer/moq

package mock

import (
	"context"
	"github.com/ONSdigital/dp-api-clients-go/health"
	"github.com/ONSdigital/dp-image-api/api"
	"github.com/ONSdigital/dp-image-api/config"
	"github.com/ONSdigital/dp-image-api/service"
	"github.com/ONSdigital/dp-kafka/v2"
	"net/http"
	"sync"
)

var (
	lockInitialiserMockDoGetHTTPServer    sync.RWMutex
	lockInitialiserMockDoGetHealthCheck   sync.RWMutex
	lockInitialiserMockDoGetHealthClient  sync.RWMutex
	lockInitialiserMockDoGetKafkaProducer sync.RWMutex
	lockInitialiserMockDoGetMongoDB       sync.RWMutex
)

// Ensure, that InitialiserMock does implement Initialiser.
// If this is not the case, regenerate this file with moq.
var _ service.Initialiser = &InitialiserMock{}

// InitialiserMock is a mock implementation of service.Initialiser.
//
//     func TestSomethingThatUsesInitialiser(t *testing.T) {
//
//         // make and configure a mocked service.Initialiser
//         mockedInitialiser := &InitialiserMock{
//             DoGetHTTPServerFunc: func(bindAddr string, router http.Handler) service.HTTPServer {
// 	               panic("mock out the DoGetHTTPServer method")
//             },
//             DoGetHealthCheckFunc: func(cfg *config.Config, buildTime string, gitCommit string, version string) (service.HealthChecker, error) {
// 	               panic("mock out the DoGetHealthCheck method")
//             },
//             DoGetHealthClientFunc: func(name string, url string) *health.Client {
// 	               panic("mock out the DoGetHealthClient method")
//             },
//             DoGetKafkaProducerFunc: func(ctx context.Context, cfg *config.Config, topic string) (kafka.IProducer, error) {
// 	               panic("mock out the DoGetKafkaProducer method")
//             },
//             DoGetMongoDBFunc: func(ctx context.Context, cfg *config.Config) (api.MongoServer, error) {
// 	               panic("mock out the DoGetMongoDB method")
//             },
//         }
//
//         // use mockedInitialiser in code that requires service.Initialiser
//         // and then make assertions.
//
//     }
type InitialiserMock struct {
	// DoGetHTTPServerFunc mocks the DoGetHTTPServer method.
	DoGetHTTPServerFunc func(bindAddr string, router http.Handler) service.HTTPServer

	// DoGetHealthCheckFunc mocks the DoGetHealthCheck method.
	DoGetHealthCheckFunc func(cfg *config.Config, buildTime string, gitCommit string, version string) (service.HealthChecker, error)

	// DoGetHealthClientFunc mocks the DoGetHealthClient method.
	DoGetHealthClientFunc func(name string, url string) *health.Client

	// DoGetKafkaProducerFunc mocks the DoGetKafkaProducer method.
	DoGetKafkaProducerFunc func(ctx context.Context, cfg *config.Config, topic string) (kafka.IProducer, error)

	// DoGetMongoDBFunc mocks the DoGetMongoDB method.
	DoGetMongoDBFunc func(ctx context.Context, cfg *config.Config) (api.MongoServer, error)

	// calls tracks calls to the methods.
	calls struct {
		// DoGetHTTPServer holds details about calls to the DoGetHTTPServer method.
		DoGetHTTPServer []struct {
			// BindAddr is the bindAddr argument value.
			BindAddr string
			// Router is the router argument value.
			Router http.Handler
		}
		// DoGetHealthCheck holds details about calls to the DoGetHealthCheck method.
		DoGetHealthCheck []struct {
			// Cfg is the cfg argument value.
			Cfg *config.Config
			// BuildTime is the buildTime argument value.
			BuildTime string
			// GitCommit is the gitCommit argument value.
			GitCommit string
			// Version is the version argument value.
			Version string
		}
		// DoGetHealthClient holds details about calls to the DoGetHealthClient method.
		DoGetHealthClient []struct {
			// Name is the name argument value.
			Name string
			// URL is the url argument value.
			URL string
		}
		// DoGetKafkaProducer holds details about calls to the DoGetKafkaProducer method.
		DoGetKafkaProducer []struct {
			// Ctx is the ctx argument value.
			Ctx context.Context
			// Cfg is the cfg argument value.
			Cfg *config.Config
			// Topic is the topic argument value.
			Topic string
		}
		// DoGetMongoDB holds details about calls to the DoGetMongoDB method.
		DoGetMongoDB []struct {
			// Ctx is the ctx argument value.
			Ctx context.Context
			// Cfg is the cfg argument value.
			Cfg *config.Config
		}
	}
}

// DoGetHTTPServer calls DoGetHTTPServerFunc.
func (mock *InitialiserMock) DoGetHTTPServer(bindAddr string, router http.Handler) service.HTTPServer {
	if mock.DoGetHTTPServerFunc == nil {
		panic("InitialiserMock.DoGetHTTPServerFunc: method is nil but Initialiser.DoGetHTTPServer was just called")
	}
	callInfo := struct {
		BindAddr string
		Router   http.Handler
	}{
		BindAddr: bindAddr,
		Router:   router,
	}
	lockInitialiserMockDoGetHTTPServer.Lock()
	mock.calls.DoGetHTTPServer = append(mock.calls.DoGetHTTPServer, callInfo)
	lockInitialiserMockDoGetHTTPServer.Unlock()
	return mock.DoGetHTTPServerFunc(bindAddr, router)
}

// DoGetHTTPServerCalls gets all the calls that were made to DoGetHTTPServer.
// Check the length with:
//     len(mockedInitialiser.DoGetHTTPServerCalls())
func (mock *InitialiserMock) DoGetHTTPServerCalls() []struct {
	BindAddr string
	Router   http.Handler
} {
	var calls []struct {
		BindAddr string
		Router   http.Handler
	}
	lockInitialiserMockDoGetHTTPServer.RLock()
	calls = mock.calls.DoGetHTTPServer
	lockInitialiserMockDoGetHTTPServer.RUnlock()
	return calls
}

// DoGetHealthCheck calls DoGetHealthCheckFunc.
func (mock *InitialiserMock) DoGetHealthCheck(cfg *config.Config, buildTime string, gitCommit string, version string) (service.HealthChecker, error) {
	if mock.DoGetHealthCheckFunc == nil {
		panic("InitialiserMock.DoGetHealthCheckFunc: method is nil but Initialiser.DoGetHealthCheck was just called")
	}
	callInfo := struct {
		Cfg       *config.Config
		BuildTime string
		GitCommit string
		Version   string
	}{
		Cfg:       cfg,
		BuildTime: buildTime,
		GitCommit: gitCommit,
		Version:   version,
	}
	lockInitialiserMockDoGetHealthCheck.Lock()
	mock.calls.DoGetHealthCheck = append(mock.calls.DoGetHealthCheck, callInfo)
	lockInitialiserMockDoGetHealthCheck.Unlock()
	return mock.DoGetHealthCheckFunc(cfg, buildTime, gitCommit, version)
}

// DoGetHealthCheckCalls gets all the calls that were made to DoGetHealthCheck.
// Check the length with:
//     len(mockedInitialiser.DoGetHealthCheckCalls())
func (mock *InitialiserMock) DoGetHealthCheckCalls() []struct {
	Cfg       *config.Config
	BuildTime string
	GitCommit string
	Version   string
} {
	var calls []struct {
		Cfg       *config.Config
		BuildTime string
		GitCommit string
		Version   string
	}
	lockInitialiserMockDoGetHealthCheck.RLock()
	calls = mock.calls.DoGetHealthCheck
	lockInitialiserMockDoGetHealthCheck.RUnlock()
	return calls
}

// DoGetHealthClient calls DoGetHealthClientFunc.
func (mock *InitialiserMock) DoGetHealthClient(name string, url string) *health.Client {
	if mock.DoGetHealthClientFunc == nil {
		panic("InitialiserMock.DoGetHealthClientFunc: method is nil but Initialiser.DoGetHealthClient was just called")
	}
	callInfo := struct {
		Name string
		URL  string
	}{
		Name: name,
		URL:  url,
	}
	lockInitialiserMockDoGetHealthClient.Lock()
	mock.calls.DoGetHealthClient = append(mock.calls.DoGetHealthClient, callInfo)
	lockInitialiserMockDoGetHealthClient.Unlock()
	return mock.DoGetHealthClientFunc(name, url)
}

// DoGetHealthClientCalls gets all the calls that were made to DoGetHealthClient.
// Check the length with:
//     len(mockedInitialiser.DoGetHealthClientCalls())
func (mock *InitialiserMock) DoGetHealthClientCalls() []struct {
	Name string
	URL  string
} {
	var calls []struct {
		Name string
		URL  string
	}
	lockInitialiserMockDoGetHealthClient.RLock()
	calls = mock.calls.DoGetHealthClient
	lockInitialiserMockDoGetHealthClient.RUnlock()
	return calls
}

// DoGetKafkaProducer calls DoGetKafkaProducerFunc.
func (mock *InitialiserMock) DoGetKafkaProducer(ctx context.Context, cfg *config.Config, topic string) (kafka.IProducer, error) {
	if mock.DoGetKafkaProducerFunc == nil {
		panic("InitialiserMock.DoGetKafkaProducerFunc: method is nil but Initialiser.DoGetKafkaProducer was just called")
	}
	callInfo := struct {
		Ctx   context.Context
		Cfg   *config.Config
		Topic string
	}{
		Ctx:   ctx,
		Cfg:   cfg,
		Topic: topic,
	}
	lockInitialiserMockDoGetKafkaProducer.Lock()
	mock.calls.DoGetKafkaProducer = append(mock.calls.DoGetKafkaProducer, callInfo)
	lockInitialiserMockDoGetKafkaProducer.Unlock()
	return mock.DoGetKafkaProducerFunc(ctx, cfg, topic)
}

// DoGetKafkaProducerCalls gets all the calls that were made to DoGetKafkaProducer.
// Check the length with:
//     len(mockedInitialiser.DoGetKafkaProducerCalls())
func (mock *InitialiserMock) DoGetKafkaProducerCalls() []struct {
	Ctx   context.Context
	Cfg   *config.Config
	Topic string
} {
	var calls []struct {
		Ctx   context.Context
		Cfg   *config.Config
		Topic string
	}
	lockInitialiserMockDoGetKafkaProducer.RLock()
	calls = mock.calls.DoGetKafkaProducer
	lockInitialiserMockDoGetKafkaProducer.RUnlock()
	return calls
}

// DoGetMongoDB calls DoGetMongoDBFunc.
func (mock *InitialiserMock) DoGetMongoDB(ctx context.Context, cfg *config.Config) (api.MongoServer, error) {
	if mock.DoGetMongoDBFunc == nil {
		panic("InitialiserMock.DoGetMongoDBFunc: method is nil but Initialiser.DoGetMongoDB was just called")
	}
	callInfo := struct {
		Ctx context.Context
		Cfg *config.Config
	}{
		Ctx: ctx,
		Cfg: cfg,
	}
	lockInitialiserMockDoGetMongoDB.Lock()
	mock.calls.DoGetMongoDB = append(mock.calls.DoGetMongoDB, callInfo)
	lockInitialiserMockDoGetMongoDB.Unlock()
	return mock.DoGetMongoDBFunc(ctx, cfg)
}

// DoGetMongoDBCalls gets all the calls that were made to DoGetMongoDB.
// Check the length with:
//     len(mockedInitialiser.DoGetMongoDBCalls())
func (mock *InitialiserMock) DoGetMongoDBCalls() []struct {
	Ctx context.Context
	Cfg *config.Config
} {
	var calls []struct {
		Ctx context.Context
		Cfg *config.Config
	}
	lockInitialiserMockDoGetMongoDB.RLock()
	calls = mock.calls.DoGetMongoDB
	lockInitialiserMockDoGetMongoDB.RUnlock()
	return calls
}
