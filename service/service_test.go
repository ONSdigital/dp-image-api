package service_test

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"testing"

	"github.com/ONSdigital/dp-api-clients-go/health"
	"github.com/ONSdigital/dp-healthcheck/healthcheck"
	"github.com/ONSdigital/dp-image-api/api"
	apiMock "github.com/ONSdigital/dp-image-api/api/mock"
	"github.com/ONSdigital/dp-image-api/config"
	"github.com/ONSdigital/dp-image-api/service"
	serviceMock "github.com/ONSdigital/dp-image-api/service/mock"
	kafka "github.com/ONSdigital/dp-kafka/v3"
	"github.com/ONSdigital/dp-kafka/v3/kafkatest"
	"github.com/pkg/errors"
	"github.com/smartystreets/goconvey/convey"
)

var (
	ctx           = context.Background()
	testBuildTime = "BuildTime"
	testGitCommit = "GitCommit"
	testVersion   = "Version"
	errServer     = errors.New("HTTP Server error")
)

var (
	errMongoDB       = errors.New("mongoDB error")
	errKafkaProducer = errors.New("KafkaProducer error")
	errHealthcheck   = errors.New("healthCheck error")
)

var funcDoGetMongoDBErr = func(_ context.Context, _ config.MongoConfig) (api.MongoServer, error) {
	return nil, errMongoDB
}

var funcDoGetHealthcheckErr = func(_ *config.Config, _ string, _ string, _ string) (service.HealthChecker, error) {
	return nil, errHealthcheck
}

var funcDoGetHTTPServerNil = func(_ string, _ http.Handler) service.HTTPServer {
	return nil
}

func TestRunPublishing(t *testing.T) {
	convey.Convey("Having a set of mocked dependencies", t, func() {
		cfg, err := config.Get()
		convey.So(err, convey.ShouldBeNil)

		mongoDBMock := &apiMock.MongoServerMock{
			CheckerFunc: func(_ context.Context, _ *healthcheck.CheckState) error { return nil },
		}

		kafkaProducerMock := &kafkatest.IProducerMock{
			ChannelsFunc: func() *kafka.ProducerChannels {
				return &kafka.ProducerChannels{}
			},
			LogErrorsFunc: func(_ context.Context) {
				// Do nothing
			},
		}

		hcMock := &serviceMock.HealthCheckerMock{
			AddCheckFunc: func(_ string, _ healthcheck.Checker) error { return nil },
			StartFunc:    func(_ context.Context) {},
		}

		serverWg := &sync.WaitGroup{}
		serverMock := &serviceMock.HTTPServerMock{
			ListenAndServeFunc: func() error {
				serverWg.Done()
				return nil
			},
		}

		failingServerMock := &serviceMock.HTTPServerMock{
			ListenAndServeFunc: func() error {
				serverWg.Done()
				return errServer
			},
		}

		funcDoGetMongoDBOk := func(_ context.Context, _ config.MongoConfig) (api.MongoServer, error) {
			return mongoDBMock, nil
		}

		funcDoGetHealthcheckOk := func(_ *config.Config, _, _, _ string) (service.HealthChecker, error) {
			return hcMock, nil
		}

		funcDoGetHTTPServer := func(_ string, _ http.Handler) service.HTTPServer {
			return serverMock
		}

		funcDoGetFailingHTTPSerer := func(_ string, _ http.Handler) service.HTTPServer {
			return failingServerMock
		}

		funcDoGetKafkaProducerOk := func(_ context.Context, _ *config.Config, _ string) (kafka.IProducer, error) {
			return kafkaProducerMock, nil
		}

		doGetKafkaProducerErrOnTopic := func(errTopic string) func(ctx context.Context, _ *config.Config, topic string) (kafka.IProducer, error) {
			return func(_ context.Context, _ *config.Config, topic string) (kafka.IProducer, error) {
				if topic == errTopic {
					return nil, errKafkaProducer
				}
				return kafkaProducerMock, nil
			}
		}

		funcDoGetHealthClientOk := func(name string, url string) *health.Client {
			return &health.Client{
				URL:  url,
				Name: name,
			}
		}

		convey.Convey("Given that initialising mongoDB returns an error", func() {
			initMock := &serviceMock.InitialiserMock{
				DoGetHTTPServerFunc:    funcDoGetHTTPServerNil,
				DoGetMongoDBFunc:       funcDoGetMongoDBErr,
				DoGetKafkaProducerFunc: funcDoGetKafkaProducerOk,
				DoGetHealthClientFunc:  funcDoGetHealthClientOk,
			}
			svcErrors := make(chan error, 1)
			svcList := service.NewServiceList(initMock)
			_, err := service.Run(ctx, cfg, svcList, testBuildTime, testGitCommit, testVersion, svcErrors)

			convey.Convey("Then service Run fails with the same error and the flag is not set. No further initialisations are attempted", func() {
				convey.So(err, convey.ShouldResemble, errMongoDB)
				convey.So(svcList.MongoDB, convey.ShouldBeFalse)
				convey.So(svcList.KafkaProducerUploaded, convey.ShouldBeFalse)
				convey.So(svcList.KafkaProducerPublished, convey.ShouldBeFalse)
				convey.So(svcList.HealthCheck, convey.ShouldBeFalse)
			})
		})

		convey.Convey("Given that initialising kafka image-uploaded producer returns an error", func() {
			initMock := &serviceMock.InitialiserMock{
				DoGetHTTPServerFunc:    funcDoGetHTTPServerNil,
				DoGetMongoDBFunc:       funcDoGetMongoDBOk,
				DoGetKafkaProducerFunc: doGetKafkaProducerErrOnTopic(cfg.ImageUploadedTopic),
				DoGetHealthClientFunc:  funcDoGetHealthClientOk,
			}
			svcErrors := make(chan error, 1)
			svcList := service.NewServiceList(initMock)
			_, err := service.Run(ctx, cfg, svcList, testBuildTime, testGitCommit, testVersion, svcErrors)

			convey.Convey("Then service Run fails with the same error and the flag is not set. No further initialisations are attempted", func() {
				convey.So(err, convey.ShouldResemble, errKafkaProducer)
				convey.So(svcList.MongoDB, convey.ShouldBeTrue)
				convey.So(svcList.KafkaProducerUploaded, convey.ShouldBeFalse)
				convey.So(svcList.KafkaProducerPublished, convey.ShouldBeFalse)
				convey.So(svcList.HealthCheck, convey.ShouldBeFalse)
			})
		})

		convey.Convey("Given that initialising kafka static-file-published producer returns an error", func() {
			initMock := &serviceMock.InitialiserMock{
				DoGetHTTPServerFunc:    funcDoGetHTTPServerNil,
				DoGetMongoDBFunc:       funcDoGetMongoDBOk,
				DoGetKafkaProducerFunc: doGetKafkaProducerErrOnTopic(cfg.StaticFilePublishedTopic),
				DoGetHealthClientFunc:  funcDoGetHealthClientOk,
			}
			svcErrors := make(chan error, 1)
			svcList := service.NewServiceList(initMock)
			_, err := service.Run(ctx, cfg, svcList, testBuildTime, testGitCommit, testVersion, svcErrors)

			convey.Convey("Then service Run fails with the same error and the flag is not set. No further initialisations are attempted", func() {
				convey.So(err, convey.ShouldResemble, errKafkaProducer)
				convey.So(svcList.MongoDB, convey.ShouldBeTrue)
				convey.So(svcList.KafkaProducerUploaded, convey.ShouldBeTrue)
				convey.So(svcList.KafkaProducerPublished, convey.ShouldBeFalse)
				convey.So(svcList.HealthCheck, convey.ShouldBeFalse)
			})
		})

		convey.Convey("Given that initialising healthcheck returns an error", func() {
			initMock := &serviceMock.InitialiserMock{
				DoGetHTTPServerFunc:    funcDoGetHTTPServerNil,
				DoGetMongoDBFunc:       funcDoGetMongoDBOk,
				DoGetHealthCheckFunc:   funcDoGetHealthcheckErr,
				DoGetKafkaProducerFunc: funcDoGetKafkaProducerOk,
				DoGetHealthClientFunc:  funcDoGetHealthClientOk,
			}
			svcErrors := make(chan error, 1)
			svcList := service.NewServiceList(initMock)
			_, err := service.Run(ctx, cfg, svcList, testBuildTime, testGitCommit, testVersion, svcErrors)

			convey.Convey("Then service Run fails with the same error and the flag is not set", func() {
				convey.So(err, convey.ShouldResemble, errHealthcheck)
				convey.So(svcList.MongoDB, convey.ShouldBeTrue)
				convey.So(svcList.KafkaProducerUploaded, convey.ShouldBeTrue)
				convey.So(svcList.KafkaProducerPublished, convey.ShouldBeTrue)
				convey.So(svcList.HealthCheck, convey.ShouldBeFalse)
			})
		})

		convey.Convey("Given that Checkers cannot be registered", func() {
			errAddheckFail := errors.New("Error(s) registering checkers for healthcheck")
			hcMockAddFail := &serviceMock.HealthCheckerMock{
				AddCheckFunc: func(_ string, _ healthcheck.Checker) error { return errAddheckFail },
				StartFunc:    func(_ context.Context) {},
			}

			initMock := &serviceMock.InitialiserMock{
				DoGetHTTPServerFunc:    funcDoGetHTTPServerNil,
				DoGetMongoDBFunc:       funcDoGetMongoDBOk,
				DoGetKafkaProducerFunc: funcDoGetKafkaProducerOk,
				DoGetHealthCheckFunc: func(_ *config.Config, _ string, _ string, _ string) (service.HealthChecker, error) {
					return hcMockAddFail, nil
				},
				DoGetHealthClientFunc: funcDoGetHealthClientOk,
			}
			svcErrors := make(chan error, 1)
			svcList := service.NewServiceList(initMock)
			_, err := service.Run(ctx, cfg, svcList, testBuildTime, testGitCommit, testVersion, svcErrors)

			convey.Convey("Then service Run fails, but all checks try to register", func() {
				convey.So(err, convey.ShouldNotBeNil)
				convey.So(err.Error(), convey.ShouldResemble, fmt.Sprintf("unable to register checkers: %s", errAddheckFail.Error()))
				convey.So(svcList.MongoDB, convey.ShouldBeTrue)
				convey.So(svcList.HealthCheck, convey.ShouldBeTrue)
				convey.So(hcMockAddFail.AddCheckCalls(), convey.ShouldHaveLength, 4)
				convey.So(hcMockAddFail.AddCheckCalls()[0].Name, convey.ShouldResemble, "Mongo DB")
				convey.So(hcMockAddFail.AddCheckCalls()[1].Name, convey.ShouldResemble, "Uploaded Kafka Producer")
				convey.So(hcMockAddFail.AddCheckCalls()[2].Name, convey.ShouldResemble, "Published Kafka Producer")
				convey.So(hcMockAddFail.AddCheckCalls()[3].Name, convey.ShouldResemble, "Zebedee")
			})
		})

		convey.Convey("Given that all dependencies are successfully initialised", func() {
			initMock := &serviceMock.InitialiserMock{
				DoGetHTTPServerFunc:    funcDoGetHTTPServer,
				DoGetMongoDBFunc:       funcDoGetMongoDBOk,
				DoGetKafkaProducerFunc: funcDoGetKafkaProducerOk,
				DoGetHealthCheckFunc:   funcDoGetHealthcheckOk,
				DoGetHealthClientFunc:  funcDoGetHealthClientOk,
			}
			svcErrors := make(chan error, 1)
			svcList := service.NewServiceList(initMock)
			serverWg.Add(1)
			_, err := service.Run(ctx, cfg, svcList, testBuildTime, testGitCommit, testVersion, svcErrors)

			convey.Convey("Then service Run succeeds and all the flags are set", func() {
				convey.So(err, convey.ShouldBeNil)
				convey.So(svcList.MongoDB, convey.ShouldBeTrue)
				convey.So(svcList.KafkaProducerUploaded, convey.ShouldBeTrue)
				convey.So(svcList.KafkaProducerPublished, convey.ShouldBeTrue)
				convey.So(svcList.HealthCheck, convey.ShouldBeTrue)
			})

			convey.Convey("The checkers are registered and the healthcheck and http server started", func() {
				convey.So(hcMock.AddCheckCalls(), convey.ShouldHaveLength, 4)
				convey.So(hcMock.AddCheckCalls()[0].Name, convey.ShouldResemble, "Mongo DB")
				convey.So(hcMock.AddCheckCalls()[1].Name, convey.ShouldResemble, "Uploaded Kafka Producer")
				convey.So(hcMock.AddCheckCalls()[2].Name, convey.ShouldResemble, "Published Kafka Producer")
				convey.So(hcMock.AddCheckCalls()[3].Name, convey.ShouldEqual, "Zebedee")
				convey.So(initMock.DoGetHTTPServerCalls(), convey.ShouldHaveLength, 1)
				convey.So(initMock.DoGetHTTPServerCalls()[0].BindAddr, convey.ShouldEqual, "localhost:24700")
				convey.So(hcMock.StartCalls(), convey.ShouldHaveLength, 1)
				serverWg.Wait() // Wait for HTTP server go-routine to finish
				convey.So(serverMock.ListenAndServeCalls(), convey.ShouldHaveLength, 1)
			})
		})

		convey.Convey("Given that all dependencies are successfully initialised but the http server fails", func() {
			initMock := &serviceMock.InitialiserMock{
				DoGetHTTPServerFunc:    funcDoGetFailingHTTPSerer,
				DoGetMongoDBFunc:       funcDoGetMongoDBOk,
				DoGetKafkaProducerFunc: funcDoGetKafkaProducerOk,
				DoGetHealthCheckFunc:   funcDoGetHealthcheckOk,
				DoGetHealthClientFunc:  funcDoGetHealthClientOk,
			}
			svcErrors := make(chan error, 1)
			svcList := service.NewServiceList(initMock)
			serverWg.Add(1)
			_, err := service.Run(ctx, cfg, svcList, testBuildTime, testGitCommit, testVersion, svcErrors)
			convey.So(err, convey.ShouldBeNil)

			convey.Convey("Then the error is returned in the error channel", func() {
				sErr := <-svcErrors
				convey.So(sErr.Error(), convey.ShouldResemble, fmt.Sprintf("failure in http listen and serve: %s", errServer.Error()))
				convey.So(failingServerMock.ListenAndServeCalls(), convey.ShouldHaveLength, 1)
			})
		})

		convey.Convey("Given that all required dependencies are successfully initialised in web mode", func() {
			cfg.IsPublishing = false
			initMock := &serviceMock.InitialiserMock{
				DoGetHTTPServerFunc:  funcDoGetHTTPServer,
				DoGetMongoDBFunc:     funcDoGetMongoDBOk,
				DoGetHealthCheckFunc: funcDoGetHealthcheckOk,
			}
			svcErrors := make(chan error, 1)
			svcList := service.NewServiceList(initMock)
			serverWg.Add(1)
			_, err := service.Run(ctx, cfg, svcList, testBuildTime, testGitCommit, testVersion, svcErrors)

			convey.Convey("Then service Run succeeds but only the required flags are set", func() {
				convey.So(err, convey.ShouldBeNil)
				convey.So(svcList.MongoDB, convey.ShouldBeTrue)
				convey.So(svcList.KafkaProducerUploaded, convey.ShouldBeFalse)
				convey.So(svcList.KafkaProducerPublished, convey.ShouldBeFalse)
				convey.So(svcList.HealthCheck, convey.ShouldBeTrue)
			})
		})
	})
}

func TestClose(t *testing.T) {
	convey.Convey("Having a correctly initialised service", t, func() {
		cfg, err := config.Get()
		convey.So(err, convey.ShouldBeNil)

		hcStopped := false
		serverStopped := false
		mongoStopped := false

		// healthcheck Stop does not depend on any other service being closed/stopped
		hcMock := &serviceMock.HealthCheckerMock{
			AddCheckFunc: func(_ string, _ healthcheck.Checker) error { return nil },
			StartFunc:    func(_ context.Context) {},
			StopFunc:     func() { hcStopped = true },
		}

		// server Shutdown will fail if healthcheck is not stopped
		serverMock := &serviceMock.HTTPServerMock{
			ListenAndServeFunc: func() error { return nil },
			ShutdownFunc: func(_ context.Context) error {
				if !hcStopped {
					return errors.New("Server stopped before healthcheck")
				}
				serverStopped = true
				return nil
			},
		}

		// mongoDB Close will fail if healthcheck and http server are not already closed
		mongoDBMock := &apiMock.MongoServerMock{
			CheckerFunc: func(_ context.Context, _ *healthcheck.CheckState) error { return nil },
			CloseFunc: func(_ context.Context) error {
				if !hcStopped || !serverStopped {
					return errors.New("MongoDB closed before stopping healthcheck or HTTP server")
				}
				mongoStopped = true
				return nil
			},
		}

		// kafkaProducerMock (for any kafka producer) will fail if healthcheck, http server and mongo are not already closed
		createKafkaProducerMock := func() *kafkatest.IProducerMock {
			return &kafkatest.IProducerMock{
				CheckerFunc: func(_ context.Context, _ *healthcheck.CheckState) error { return nil },
				CloseFunc: func(_ context.Context) error {
					if !hcStopped || !serverStopped || !mongoStopped {
						return errors.New("KafkaProducer closed before stopping healthcheck, MongoDB or HTTP server")
					}
					return nil
				},
				LogErrorsFunc: func(_ context.Context) {
					// Do nothing
				},
				ChannelsFunc: kafka.CreateProducerChannels,
			}
		}
		kafkaUploadedProducerMock := createKafkaProducerMock()
		kafkaPublishedProducerMock := createKafkaProducerMock()
		doGetKafkaProducerFunc := func(_ context.Context, cfg *config.Config, topic string) (kafka.IProducer, error) {
			if topic == cfg.ImageUploadedTopic {
				return kafkaUploadedProducerMock, nil
			} else if topic == cfg.StaticFilePublishedTopic {
				return kafkaPublishedProducerMock, nil
			}
			return nil, errors.New("wrong topic")
		}

		convey.Convey("Closing the service results in all the dependencies being closed in the expected order", func() {
			initMock := &serviceMock.InitialiserMock{
				DoGetHTTPServerFunc:    func(_ string, _ http.Handler) service.HTTPServer { return serverMock },
				DoGetMongoDBFunc:       func(_ context.Context, _ config.MongoConfig) (api.MongoServer, error) { return mongoDBMock, nil },
				DoGetKafkaProducerFunc: doGetKafkaProducerFunc,
				DoGetHealthCheckFunc: func(_ *config.Config, _ string, _ string, _ string) (service.HealthChecker, error) {
					return hcMock, nil
				},
				DoGetHealthClientFunc: func(_, _ string) *health.Client { return &health.Client{} },
			}

			svcErrors := make(chan error, 1)
			svcList := service.NewServiceList(initMock)
			svc, err := service.Run(ctx, cfg, svcList, testBuildTime, testGitCommit, testVersion, svcErrors)
			convey.So(err, convey.ShouldBeNil)

			err = svc.Close(context.Background())
			convey.So(err, convey.ShouldBeNil)
			convey.So(hcMock.StopCalls(), convey.ShouldHaveLength, 1)
			convey.So(serverMock.ShutdownCalls(), convey.ShouldHaveLength, 1)
			convey.So(mongoDBMock.CloseCalls(), convey.ShouldHaveLength, 1)
			convey.So(kafkaUploadedProducerMock.CloseCalls(), convey.ShouldHaveLength, 1)
			convey.So(kafkaPublishedProducerMock.CloseCalls(), convey.ShouldHaveLength, 1)
		})

		convey.Convey("If services fail to stop, the Close operation tries to close all dependencies and returns an error", func() {
			failingserverMock := &serviceMock.HTTPServerMock{
				ListenAndServeFunc: func() error { return nil },
				ShutdownFunc: func(_ context.Context) error {
					return errors.New("Failed to stop http server")
				},
			}

			initMock := &serviceMock.InitialiserMock{
				DoGetHTTPServerFunc:    func(_ string, _ http.Handler) service.HTTPServer { return failingserverMock },
				DoGetMongoDBFunc:       func(_ context.Context, _ config.MongoConfig) (api.MongoServer, error) { return mongoDBMock, nil },
				DoGetKafkaProducerFunc: doGetKafkaProducerFunc,
				DoGetHealthCheckFunc: func(_ *config.Config, _, _, _ string) (service.HealthChecker, error) {
					return hcMock, nil
				},
				DoGetHealthClientFunc: func(_, _ string) *health.Client { return &health.Client{} },
			}

			svcErrors := make(chan error, 1)
			svcList := service.NewServiceList(initMock)
			svc, err := service.Run(ctx, cfg, svcList, testBuildTime, testGitCommit, testVersion, svcErrors)
			convey.So(err, convey.ShouldBeNil)

			err = svc.Close(context.Background())
			convey.So(err, convey.ShouldNotBeNil)
			convey.So(hcMock.StopCalls(), convey.ShouldHaveLength, 1)
			convey.So(failingserverMock.ShutdownCalls(), convey.ShouldHaveLength, 1)
			convey.So(mongoDBMock.CloseCalls(), convey.ShouldHaveLength, 1)
			convey.So(kafkaUploadedProducerMock.CloseCalls(), convey.ShouldHaveLength, 1)
			convey.So(kafkaPublishedProducerMock.CloseCalls(), convey.ShouldHaveLength, 1)
		})
	})
}
