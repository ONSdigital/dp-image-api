package mongo

import (
	"context"
	"errors"

	"github.com/ONSdigital/dp-healthcheck/healthcheck"
	mongolib "github.com/ONSdigital/dp-mongodb"
	mongoHealth "github.com/ONSdigital/dp-mongodb/health"
	"github.com/globalsign/mgo"
)

// Mongo represents a simplistic MongoDB configuration.
type Mongo struct {
	Collection   string
	Database     string
	Session      *mgo.Session
	URI          string
	client       *mongoHealth.Client
	healthClient *mongoHealth.CheckMongoClient
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
	m.client = mongoHealth.NewClient(m.Session)
	m.healthClient = &mongoHealth.CheckMongoClient{
		Client:      *m.client,
		Healthcheck: m.client.Healthcheck,
	}
	return nil
}

// Close closes the mongo session and returns any error
func (m *Mongo) Close(ctx context.Context) error {
	return mongolib.Close(ctx, m.Session)
}

// Checker is called by the healthcheck library to check the health state of this mongoDB instance
func (m *Mongo) Checker(ctx context.Context, state *healthcheck.CheckState) error {
	return m.healthClient.Checker(ctx, state)
}
