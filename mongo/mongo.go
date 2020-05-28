package mongo

import (
	"context"
	"errors"
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

	// Filter by collectionID
	colIDFilter := make(bson.M)
	colIDFilter["collection_id"] = collectionID

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
	update := bson.M{"$set": updates, "$setOnInsert": bson.M{"last_updated": time.Now()}}
	if err := s.DB(m.Database).C(imagesCol).UpdateId(id, update); err != nil {
		if err == mgo.ErrNotFound {
			return errs.ErrImageNotFound
		}
		return err
	}

	return nil
}

func createImageUpdateQuery(ctx context.Context, id string, image *models.Image) bson.M {
	updates := make(bson.M)

	log.Event(ctx, "building update query for image resource", log.INFO, log.INFO, log.Data{"image_id": id, "image": image, "updates": updates})

	updates["collection_id"] = image.CollectionID
	updates["state"] = image.State
	updates["filename"] = image.Filename
	updates["type"] = image.Type

	if image.License != nil {
		license := make(bson.M)
		license["title"] = image.License.Title
		license["href"] = image.License.Href
		updates["license"] = license
	} else {
		updates["license"] = nil
	}

	if image.Upload != nil {
		upload := make(bson.M)
		upload["path"] = image.Upload.Path
		updates["upload"] = upload
	} else {
		updates["upload"] = nil
	}

	if image.Downloads != nil {
		variants := make(bson.M)
		for variantKey, variant := range image.Downloads {
			resolutions := make(bson.M)
			for resolutionKey, download := range variant {
				d := make(bson.M)
				d["size"] = download.Size
				d["href"] = download.Href
				d["public"] = download.Public
				d["private"] = download.Private
				resolutions[resolutionKey] = d
			}
			variants[variantKey] = resolutions
		}
		updates["downloads"] = variants
	} else {
		updates["downloads"] = nil
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
