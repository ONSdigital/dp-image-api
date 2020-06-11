package mongo

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/ONSdigital/dp-healthcheck/healthcheck"
	errs "github.com/ONSdigital/dp-image-api/apierrors"
	"github.com/ONSdigital/dp-image-api/models"
	dpMongodb "github.com/ONSdigital/dp-mongodb"
	dpMongoHealth "github.com/ONSdigital/dp-mongodb/health"
	"github.com/ONSdigital/log.go/log"
	"github.com/globalsign/mgo"
	"github.com/globalsign/mgo/bson"
)

const imagesCol = "images"

// Mongo represents a simplistic MongoDB configuration.
type Mongo struct {
	Collection   string
	Database     string
	Session      *mgo.Session
	URI          string
	client       *dpMongoHealth.Client
	healthClient *dpMongoHealth.CheckMongoClient
}

// Init creates a new mgo.Session with a strong consistency and a write mode of "majority".
func (m *Mongo) Init() (err error) {
	if m.Session != nil {
		return errors.New("session already exists")
	}

	// Create session
	if m.Session, err = mgo.Dial(m.URI); err != nil {
		return err
	}
	m.Session.EnsureSafe(&mgo.Safe{WMode: "majority"})
	m.Session.SetMode(mgo.Strong, true)

	// Create client and healthclient from session
	m.client = dpMongoHealth.NewClient(m.Session)
	m.healthClient = &dpMongoHealth.CheckMongoClient{
		Client:      *m.client,
		Healthcheck: m.client.Healthcheck,
	}
	return nil
}

// Close closes the mongo session and returns any error
func (m *Mongo) Close(ctx context.Context) error {
	return dpMongodb.Close(ctx, m.Session)
}

// Checker is called by the healthcheck library to check the health state of this mongoDB instance
func (m *Mongo) Checker(ctx context.Context, state *healthcheck.CheckState) error {
	return m.healthClient.Checker(ctx, state)
}

// GetImages retrieves all images documents corresponding to the provided collectionID
func (m *Mongo) GetImages(ctx context.Context, collectionID string) ([]models.Image, error) {
	s := m.Session.Copy()
	defer s.Close()
	log.Event(ctx, "getting images for collectionID", log.Data{"collectionID": collectionID})

	// Filter by collectionID, if provided
	colIDFilter := make(bson.M)
	if collectionID != "" {
		colIDFilter["collection_id"] = collectionID
	}

	iter := s.DB(m.Database).C(imagesCol).Find(colIDFilter).Iter()
	defer func() {
		err := iter.Close()
		if err != nil {
			log.Event(ctx, "error closing iterator", log.ERROR, log.Error(err))
		}
	}()

	results := []models.Image{}
	if err := iter.All(&results); err != nil {
		if err == mgo.ErrNotFound {
			return nil, errs.ErrImageNotFound
		}
		return nil, err
	}

	return results, nil
}

// GetImage retrieves an image document by its ID
func (m *Mongo) GetImage(ctx context.Context, id string) (*models.Image, error) {
	s := m.Session.Copy()
	defer s.Close()
	log.Event(ctx, "getting image by ID", log.Data{"id": id})

	var image models.Image
	err := s.DB(m.Database).C(imagesCol).Find(bson.M{"_id": id}).One(&image)
	if err != nil {
		if err == mgo.ErrNotFound {
			return nil, errs.ErrImageNotFound
		}
		return nil, err
	}

	return &image, nil
}

// UpdateImage updates an existing image document
func (m *Mongo) UpdateImage(ctx context.Context, id string, image *models.Image) error {
	s := m.Session.Copy()
	defer s.Close()
	log.Event(ctx, "updating image", log.Data{"id": id})

	updates := createImageUpdateQuery(ctx, id, image)
	if len(updates) == 0 {
		log.Event(ctx, "nothing to update")
		return nil
	}

	update := bson.M{"$set": updates, "$setOnInsert": bson.M{"last_updated": time.Now()}}
	if err := s.DB(m.Database).C(imagesCol).UpdateId(id, update); err != nil {
		if err == mgo.ErrNotFound {
			return errs.ErrImageNotFound
		}
		return err
	}

	return nil
}

// createImageUpdateQuery generates the bson model to update an image with the provided image update.
// Fields present in mongoDB will not be deleted if they are not present in the image update object.
func createImageUpdateQuery(ctx context.Context, id string, image *models.Image) bson.M {
	updates := make(bson.M)

	log.Event(ctx, "building update query for image resource", log.INFO, log.INFO, log.Data{"image_id": id, "image": image, "updates": updates})

	if image.CollectionID != "" {
		updates["collection_id"] = image.CollectionID
	}
	if image.State != "" {
		updates["state"] = image.State
	}
	if image.Filename != "" {
		updates["filename"] = image.Filename
	}
	if image.Type != "" {
		updates["type"] = image.Type
	}

	if image.License != nil {
		if image.License.Title != "" {
			updates["license.title"] = image.License.Title
		}
		if image.License.Href != "" {
			updates["license.href"] = image.License.Href
		}
	}

	if image.Upload != nil {
		if image.Upload.Path != "" {
			updates["upload"] = image.Upload
		}
	}

	if image.Downloads != nil {
		for variantKey, variant := range image.Downloads {
			for resolutionKey, download := range variant {
				if download.Size != nil {
					updates[fmt.Sprintf("downloads.%s.%s.size", variantKey, resolutionKey)] = download.Size
				}
				if download.Href != "" {
					updates[fmt.Sprintf("downloads.%s.%s.href", variantKey, resolutionKey)] = download.Href
				}
				if download.Public != "" {
					updates[fmt.Sprintf("downloads.%s.%s.public", variantKey, resolutionKey)] = download.Public
				}
				if download.Private != "" {
					updates[fmt.Sprintf("downloads.%s.%s.private", variantKey, resolutionKey)] = download.Private
				}
			}
		}
	}

	return updates
}

// UpsertImage adds or overides an existing image document
func (m *Mongo) UpsertImage(ctx context.Context, id string, image *models.Image) (err error) {
	s := m.Session.Copy()
	defer s.Close()
	log.Event(ctx, "upserting image", log.Data{"id": id})

	update := bson.M{
		"$set": image,
		"$setOnInsert": bson.M{
			"last_updated": time.Now(),
		},
	}

	_, err = s.DB(m.Database).C(imagesCol).UpsertId(id, update)
	return
}
