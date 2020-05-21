package api

import (
	"context"

	"github.com/ONSdigital/dp-healthcheck/healthcheck"
	"github.com/ONSdigital/dp-image-api/models"
)

//go:generate moq -out mock/mongo.go -pkg mock . MongoServer

// MongoServer defines the required methods from MongoDB
type MongoServer interface {
	Close(ctx context.Context) error
	Checker(ctx context.Context, state *healthcheck.CheckState) error
	GetImages(ctx context.Context, collectionID string) ([]models.Image, error)
	GetImage(id string) (*models.Image, error)
	UpdateImage(ctx context.Context, id string, image *models.Image) error
	UpsertImage(id string, image *models.Image) (err error)
}
