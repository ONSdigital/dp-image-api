package service

import (
	"context"
	"net/http"

	"github.com/ONSdigital/dp-healthcheck/healthcheck"
	"github.com/ONSdigital/dp-image-api/config"
)

//go:generate moq -out mock/initialiser.go -pkg mock . Initialiser
//go:generate moq -out mock/mongo.go -pkg mock . IMongo
//go:generate moq -out mock/server.go -pkg mock . IServer
//go:generate moq -out mock/healthCheck.go -pkg mock . IHealthCheck

// Initialiser defines the methods to initialise external services
type Initialiser interface {
	DoGetHTTPServer(bindAddr string, router http.Handler) IServer
	DoGetMongoDB(ctx context.Context, cfg *config.Config) (IMongo, error)
	DoGetHealthCheck(cfg *config.Config, buildTime, gitCommit, version string) (IHealthCheck, error)
}

// IMongo defines the required methods from MongoDB
type IMongo interface {
	Close(ctx context.Context) error
	Checker(ctx context.Context, state *healthcheck.CheckState) error
}

// IServer defines the required methods from the HTTP server
type IServer interface {
	ListenAndServe() error
	Shutdown(ctx context.Context) error
}

// IHealthCheck defines the required methods from Healthcheck
type IHealthCheck interface {
	Handler(w http.ResponseWriter, req *http.Request)
	Start(ctx context.Context)
	Stop()
	AddCheck(name string, checker healthcheck.Checker) (err error)
}
