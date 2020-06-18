package api

import (
	"context"
	"net/http"

	dpauth "github.com/ONSdigital/dp-authorisation/auth"
	"github.com/ONSdigital/dp-healthcheck/healthcheck"
	"github.com/ONSdigital/dp-image-api/models"
)

//go:generate moq -out mock/mongo.go -pkg mock . MongoServer
//go:generate moq -out mock/auth.go -pkg mock . AuthHandler

// MongoServer defines the required methods from MongoDB
type MongoServer interface {
	Close(ctx context.Context) error
	Checker(ctx context.Context, state *healthcheck.CheckState) (err error)
	GetImages(ctx context.Context, collectionID string) (images []models.Image, err error)
	GetImage(ctx context.Context, id string) (image *models.Image, err error)
	UpdateImage(ctx context.Context, id string, image *models.Image) (didChange bool, err error)
	UpsertImage(ctx context.Context, id string, image *models.Image) (err error)
}

// AuthHandler interface for adding auth to endpoints
type AuthHandler interface {
	Require(required dpauth.Permissions, handler http.HandlerFunc) http.HandlerFunc
}
