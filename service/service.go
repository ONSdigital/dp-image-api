package service

import (
	"context"

	"github.com/ONSdigital/dp-api-clients-go/health"
	dpauth "github.com/ONSdigital/dp-authorisation/auth"
	"github.com/ONSdigital/dp-image-api/api"
	"github.com/ONSdigital/dp-image-api/config"
	kafka "github.com/ONSdigital/dp-kafka"
	"github.com/ONSdigital/dp-net/handlers"
	"github.com/ONSdigital/log.go/log"
	"github.com/gorilla/mux"
	"github.com/justinas/alice"
	"github.com/pkg/errors"
)

// Service contains all the configs, server and clients to run the Image API
type Service struct {
	config        *config.Config
	server        HTTPServer
	router        *mux.Router
	api           *api.API
	serviceList   *ExternalServiceList
	healthCheck   HealthChecker
	mongoDB       api.MongoServer
	kafkaProducer kafka.IProducer
}

// Run the service
func Run(ctx context.Context, serviceList *ExternalServiceList, buildTime, gitCommit, version string, svcErrors chan error) (*Service, error) {
	log.Event(ctx, "running service", log.INFO)

	// Read config
	cfg, err := config.Get()
	if err != nil {
		return nil, errors.Wrap(err, "unable to retrieve service configuration")
	}
	log.Event(ctx, "got service configuration", log.Data{"config": cfg}, log.INFO)

	// Get HTTP Server with collectionID checkHeader middleware
	r := mux.NewRouter()
	middleware := alice.New(handlers.CheckHeader(handlers.CollectionID))
	s := serviceList.GetHTTPServer(cfg.BindAddr, middleware.Then(r))

	// Get Health client for Zebedee and permissions only if we are in publishing mode
	var zc *health.Client
	var auth api.AuthHandler
	if cfg.IsPublishing {
		zc = serviceList.GetHealthClient("Zebedee", cfg.ZebedeeURL)
		auth = getAuthorisationHandlers(zc)
	}

	// Get MongoDB client
	mongoDB, err := serviceList.GetMongoDB(ctx, cfg)
	if err != nil {
		log.Event(ctx, "failed to initialise mongo DB", log.FATAL, log.Error(err))
		return nil, err
	}

	// Get Kafka producer
	kafkaProducer, err := serviceList.GetKafkaProducer(ctx, cfg)
	if err != nil {
		log.Event(ctx, "failed to create a kafka producer", log.FATAL, log.Error(err))
		return nil, err
	}

	// Setup the API
	a := api.Setup(ctx, cfg, r, auth, mongoDB, kafkaProducer)

	// Get HealthCheck
	hc, err := serviceList.GetHealthCheck(cfg, buildTime, gitCommit, version)
	if err != nil {
		log.Event(ctx, "could not instantiate healthcheck", log.FATAL, log.Error(err))
		return nil, err
	}
	if err := registerCheckers(ctx, cfg, hc, mongoDB, kafkaProducer, zc); err != nil {
		return nil, errors.Wrap(err, "unable to register checkers")
	}

	r.StrictSlash(true).Path("/health").HandlerFunc(hc.Handler)
	hc.Start(ctx)

	// Run the http server in a new go-routine
	go func() {
		if err := s.ListenAndServe(); err != nil {
			svcErrors <- errors.Wrap(err, "failure in http listen and serve")
		}
	}()

	return &Service{
		config:        cfg,
		server:        s,
		router:        r,
		api:           a,
		serviceList:   serviceList,
		healthCheck:   hc,
		mongoDB:       mongoDB,
		kafkaProducer: kafkaProducer,
	}, nil
}

// Close gracefully shuts the service down in the required order, with timeout
func (svc *Service) Close(ctx context.Context) error {
	timeout := svc.config.GracefulShutdownTimeout
	log.Event(ctx, "commencing graceful shutdown", log.Data{"graceful_shutdown_timeout": timeout}, log.INFO)
	ctx, cancel := context.WithTimeout(ctx, timeout)

	// track shutown gracefully closes up
	var gracefulShutdown bool

	go func() {
		defer cancel()
		var hasShutdownError bool

		// stop healthcheck, as it depends on everything else
		if svc.serviceList.HealthCheck {
			svc.healthCheck.Stop()
		}

		// stop any incoming requests before closing any outbound connections
		if err := svc.server.Shutdown(ctx); err != nil {
			log.Event(ctx, "failed to shutdown http server", log.Error(err), log.ERROR)
			hasShutdownError = true
		}

		// close API
		if err := svc.api.Close(ctx); err != nil {
			log.Event(ctx, "error closing API", log.Error(err), log.ERROR)
			hasShutdownError = true
		}

		// close mongoDB
		if svc.serviceList.MongoDB {
			if err := svc.mongoDB.Close(ctx); err != nil {
				log.Event(ctx, "error closing mongoDB", log.Error(err), log.ERROR)
				hasShutdownError = true
			}
		}

		// close kafka producer
		if svc.serviceList.KafkaProducer {
			if err := svc.kafkaProducer.Close(ctx); err != nil {
				log.Event(ctx, "error closing Kafka Producer", log.Error(err), log.ERROR)
				hasShutdownError = true
			}
		}

		if !hasShutdownError {
			gracefulShutdown = true
		}
	}()

	// wait for shutdown success (via cancel) or failure (timeout)
	<-ctx.Done()

	if !gracefulShutdown {
		err := errors.New("failed to shutdown gracefully")
		log.Event(ctx, "failed to shutdown gracefully ", log.ERROR, log.Error(err))
		return err
	}

	log.Event(ctx, "graceful shutdown was successful", log.INFO)
	return nil
}

func registerCheckers(ctx context.Context,
	cfg *config.Config,
	hc HealthChecker,
	mongoDB api.MongoServer,
	kafkaProducer kafka.IProducer,
	zebedeeClient *health.Client) (err error) {

	hasErrors := false

	if err = hc.AddCheck("Mongo DB", mongoDB.Checker); err != nil {
		hasErrors = true
		log.Event(ctx, "error adding check for mongo db", log.ERROR, log.Error(err))
	}

	if err = hc.AddCheck("Kafka Producer", kafkaProducer.Checker); err != nil {
		hasErrors = true
		log.Event(ctx, "error adding check for kafka producer", log.ERROR, log.Error(err))
	}

	if cfg.IsPublishing {
		if err = hc.AddCheck("Zebedee", zebedeeClient.Checker); err != nil {
			hasErrors = true
			log.Event(ctx, "error adding check for zebedeee", log.ERROR, log.Error(err))
		}
	}

	if hasErrors {
		return errors.New("Error(s) registering checkers for healthcheck")
	}
	return nil
}

// generate permissions from dp-auth-api, using the provided health client, reusing its http Client
func getAuthorisationHandlers(zc *health.Client) api.AuthHandler {
	dpauth.LoggerNamespace("dp-image-api-auth")

	log.Event(nil, "getting Authorisation Handlers", log.Data{"zc_url": zc.URL})

	authClient := dpauth.NewPermissionsClient(zc.Client)
	authVerifier := dpauth.DefaultPermissionsVerifier()

	// for checking caller permissions when we only have a user/service token
	permissions := dpauth.NewHandler(
		dpauth.NewPermissionsRequestBuilder(zc.URL),
		authClient,
		authVerifier,
	)

	return permissions
}
