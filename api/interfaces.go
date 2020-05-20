package api

import (
	"context"

	"github.com/ONSdigital/dp-healthcheck/healthcheck"
)

//go:generate moq -out mock/mongo.go -pkg mock . MongoServer

// MongoServer defines the required methods from MongoDB
type MongoServer interface {
	Close(ctx context.Context) error
	Checker(ctx context.Context, state *healthcheck.CheckState) error
}
