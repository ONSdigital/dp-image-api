package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/ONSdigital/dp-image-api/config"
	"github.com/ONSdigital/dp-image-api/service"
	"github.com/ONSdigital/log.go/v2/log"
	"github.com/pkg/errors"
)

const serviceName = "dp-image-api"

var (
	// BuildTime represents the time in which the service was built
	BuildTime string
	// GitCommit represents the commit (SHA-1) hash of the service that is running
	GitCommit string
	// Version represents the version of the service that is running
	Version string
)

func main() {
	log.Namespace = serviceName
	ctx := context.Background()

	if err := run(ctx); err != nil {
		log.Fatal(ctx, "fatal runtime error", err)
	}
}

func run(ctx context.Context) error {
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM)

	// Run the service, providing an error channel for fatal errors
	svcErrors := make(chan error, 1)
	svcList := service.NewServiceList(&service.Init{})

	// Read config
	cfg, err := config.Get()
	if err != nil {
		return errors.Wrap(err, "unable to retrieve service configuration")
	}

	svc, err := service.Run(ctx, cfg, svcList, BuildTime, GitCommit, Version, svcErrors)
	if err != nil {
		return errors.Wrap(err, "running service failed")
	}

	// blocks until an os interrupt or a fatal error occurs
	select {
	case err := <-svcErrors:
		log.Error(ctx, "service error received", err)
	case sig := <-signals:
		log.Info(ctx, "os signal received", log.Data{"signal": sig})
	}
	return svc.Close(ctx)
}
