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
	dpMongoLock "github.com/ONSdigital/dp-mongodb/dplock"
	dpMongoHealth "github.com/ONSdigital/dp-mongodb/health"
	"github.com/ONSdigital/log.go/log"
	"github.com/globalsign/mgo"
	"github.com/globalsign/mgo/bson"
)

// images collection name
const imagesCol = "images"

// locked images collection name
const imagesLockCol = "images_locks"

// Mongo represents a simplistic MongoDB configuration, with session, health and lock clients
type Mongo struct {
	Collection   string
	Database     string
	Session      *mgo.Session
	URI          string
	client       *dpMongoHealth.Client
	healthClient *dpMongoHealth.CheckMongoClient
	lockClient   *dpMongoLock.Lock
}

// Init creates a new mgo.Session with a strong consistency and a write mode of "majority".
func (m *Mongo) Init(ctx context.Context) (err error) {
	if m.Session != nil {
		return errors.New("session already exists")
	}

	// Create session
	if m.Session, err = mgo.Dial(m.URI); err != nil {
		return err
	}
	m.Session.EnsureSafe(&mgo.Safe{WMode: "majority"})
	m.Session.SetMode(mgo.Strong, true)

	databaseCollectionBuilder := make(map[dpMongoHealth.Database][]dpMongoHealth.Collection)
	databaseCollectionBuilder[(dpMongoHealth.Database)(m.Database)] = []dpMongoHealth.Collection{(dpMongoHealth.Collection)(m.Collection), (dpMongoHealth.Collection)(imagesLockCol)}
	// Create client and healthclient from session
	m.client = dpMongoHealth.NewClientWithCollections(m.Session, databaseCollectionBuilder)
	m.healthClient = &dpMongoHealth.CheckMongoClient{
		Client:      *m.client,
		Healthcheck: m.client.Healthcheck,
	}

	// Create MongoDB lock client, which also starts the purger loop
	m.lockClient = dpMongoLock.New(ctx, m.Session, m.Database, imagesCol)
	return nil
}

// AcquireImageLock tries to lock the provided imageID.
// If the image is already locked, this function will block until it's released,
// at which point we acquire the lock and return.
func (m *Mongo) AcquireImageLock(ctx context.Context, imageID string) (lockID string, err error) {
	return m.lockClient.Acquire(ctx, imageID)
}

// UnlockImage releases an exclusive mongoDB lock for the provided lockId (if it exists)
func (m *Mongo) UnlockImage(lockID string) error {
	return m.lockClient.Unlock(lockID)
}

// Close closes the mongo session and returns any error
func (m *Mongo) Close(ctx context.Context) error {
	m.lockClient.Close(ctx)
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
func (m *Mongo) UpdateImage(ctx context.Context, id string, image *models.Image) (bool, error) {
	s := m.Session.Copy()
	defer s.Close()
	log.Event(ctx, "updating image", log.Data{"id": id})

	updates := createImageUpdateQuery(ctx, id, image)
	if len(updates) == 0 {
		log.Event(ctx, "nothing to update")
		return false, nil
	}

	update := bson.M{"$set": updates, "$setOnInsert": bson.M{"last_updated": time.Now()}}
	if err := s.DB(m.Database).C(imagesCol).UpdateId(id, update); err != nil {
		if err == mgo.ErrNotFound {
			return false, errs.ErrImageNotFound
		}
		return false, err
	}

	return true, nil
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
		for variant, download := range image.Downloads {
			if download.ID != "" {
				updates[fmt.Sprintf("downloads.%s.id", variant)] = download.ID
			}
			if download.Size != nil {
				updates[fmt.Sprintf("downloads.%s.size", variant)] = download.Size
			}
			if download.Type != "" {
				updates[fmt.Sprintf("downloads.%s.type", variant)] = download.Type
			}
			if download.Width != nil {
				updates[fmt.Sprintf("downloads.%s.width", variant)] = download.Width
			}
			if download.Height != nil {
				updates[fmt.Sprintf("downloads.%s.height", variant)] = download.Height
			}
			if download.Links != nil {
				updates[fmt.Sprintf("downloads.%s.links", variant)] = download.Links
			}
			if download.Private != "" {
				updates[fmt.Sprintf("downloads.%s.private_bucket", variant)] = download.Private
			}
			if download.State != "" {
				updates[fmt.Sprintf("downloads.%s.state", variant)] = download.State
			}
			if download.Error != "" {
				updates[fmt.Sprintf("downloads.%s.error", variant)] = download.Error
			}
			if download.ImportStarted != nil {
				updates[fmt.Sprintf("downloads.%s.import_started", variant)] = download.ImportStarted
			}
			if download.ImportCompleted != nil {
				updates[fmt.Sprintf("downloads.%s.import_completed", variant)] = download.ImportCompleted
			}
			if download.PublishStarted != nil {
				updates[fmt.Sprintf("downloads.%s.publish_started", variant)] = download.PublishStarted
			}
			if download.PublishCompleted != nil {
				updates[fmt.Sprintf("downloads.%s.publish_completed", variant)] = download.PublishCompleted
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
