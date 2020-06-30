package api

import (
	"context"
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"

	dpauth "github.com/ONSdigital/dp-authorisation/auth"
	"github.com/ONSdigital/dp-image-api/apierrors"
	"github.com/ONSdigital/dp-image-api/config"
	"github.com/ONSdigital/dp-image-api/event"
	"github.com/ONSdigital/dp-image-api/schema"
	kafka "github.com/ONSdigital/dp-kafka"
	"github.com/ONSdigital/log.go/log"
	"github.com/gorilla/mux"
)

//API provides a struct to wrap the api around
type API struct {
	Router            *mux.Router
	mongoDB           MongoServer
	auth              AuthHandler
	uploadProducer    *event.AvroProducer
	publishedProducer *event.AvroProducer
}

// Setup creates the API struct and its endpoints with corresponding handlers
func Setup(ctx context.Context, cfg *config.Config, r *mux.Router, auth AuthHandler, mongoDB MongoServer, uploadedKafkaProducer, publishedKafkaProducer kafka.IProducer) *API {

	api := &API{
		Router:  r,
		auth:    auth,
		mongoDB: mongoDB,
	}

	if cfg.IsPublishing {
		api.uploadProducer = event.NewAvroProducer(uploadedKafkaProducer.Channels().Output, schema.ImageUploadedEvent)
		api.publishedProducer = event.NewAvroProducer(publishedKafkaProducer.Channels().Output, schema.ImagePublishedEvent)
		r.HandleFunc("/images", auth.Require(dpauth.Permissions{Create: true}, api.CreateImageHandler)).Methods(http.MethodPost)
		r.HandleFunc("/images", auth.Require(dpauth.Permissions{Read: true}, api.GetImagesHandler)).Methods(http.MethodGet)
		r.HandleFunc("/images/{id}", auth.Require(dpauth.Permissions{Read: true}, api.GetImageHandler)).Methods(http.MethodGet)
		r.HandleFunc("/images/{id}", auth.Require(dpauth.Permissions{Update: true}, api.UpdateImageHandler)).Methods(http.MethodPut)
		r.HandleFunc("/images/{id}/upload", auth.Require(dpauth.Permissions{Update: true}, api.UploadImageHandler)).Methods(http.MethodPost)
		r.HandleFunc("/images/{id}/publish", auth.Require(dpauth.Permissions{Update: true}, api.PublishImageHandler)).Methods(http.MethodPost)
		r.HandleFunc("/images/{id}/downloads/{variant}/import", auth.Require(dpauth.Permissions{Update: true}, api.ImportVariantHandler)).Methods(http.MethodPost)
	} else {
		r.HandleFunc("/images", api.GetImagesHandler).Methods(http.MethodGet)
		r.HandleFunc("/images/{id}", api.GetImageHandler).Methods(http.MethodGet)
	}
	return api
}

// Close is called during graceful shutdown to give the API an opportunity to perform any required disposal task
func (*API) Close(ctx context.Context) error {
	log.Event(ctx, "graceful shutdown of api complete", log.INFO)
	return nil
}

// WriteJSONBody marshals the provided interface into json, and writes it to the response body.
func WriteJSONBody(ctx context.Context, v interface{}, w http.ResponseWriter, data log.Data) error {

	// Set headers
	w.Header().Set("Content-Type", "application/json; charset=utf-8")

	// Marshal provided model
	payload, err := json.Marshal(v)
	if err != nil {
		handleError(ctx, w, apierrors.ErrInternalServer, data)
		return err
	}

	// Write payload to body
	if _, err := w.Write(payload); err != nil {
		handleError(ctx, w, apierrors.ErrInternalServer, data)
		return err
	}
	return nil
}

// ReadJSONBody reads the bytes from the provided body, and marshals it to the provided model interface.
func ReadJSONBody(ctx context.Context, body io.ReadCloser, v interface{}, w http.ResponseWriter, data log.Data) error {
	defer body.Close()

	// Set headers
	w.Header().Set("Content-Type", "application/json; charset=utf-8")

	// Get Body bytes
	payload, err := ioutil.ReadAll(body)
	if err != nil {
		handleError(ctx, w, apierrors.ErrUnableToReadMessage, data)
		return err
	}

	// Unmarshal body bytes to model
	if err := json.Unmarshal(payload, v); err != nil {
		handleError(ctx, w, apierrors.ErrUnableToParseJSON, data)
		return err
	}

	return nil
}

// handleError is a utility function that maps api errors to an http status code and sets the provided responseWriter accordingly
func handleError(ctx context.Context, w http.ResponseWriter, err error, data log.Data) {
	var status int
	if err != nil {
		switch err {
		case apierrors.ErrImageNotFound,
			apierrors.ErrVariantNotFound:
			status = http.StatusNotFound
		case apierrors.ErrUnableToReadMessage,
			apierrors.ErrColIDMismatch,
			apierrors.ErrUnableToParseJSON,
			apierrors.ErrWrongColID,
			apierrors.ErrImageFilenameTooLong,
			apierrors.ErrImageNoCollectionID,
			apierrors.ErrImageInvalidState,
			apierrors.ErrImageIDMismatch:
			status = http.StatusBadRequest
		case apierrors.ErrResourceState,
			apierrors.ErrImageAlreadyCompleted,
			apierrors.ErrImageStateTransitionNotAllowed,
			apierrors.ErrVariantStateTransitionNotAllowed,
			apierrors.ErrImagePublishWrongEndpoint,
			apierrors.ErrImageDownloadInvalidState:
			status = http.StatusForbidden
		default:
			status = http.StatusInternalServerError
		}
	}

	if data == nil {
		data = log.Data{}
	}

	data["response_status"] = status
	log.Event(ctx, "request unsuccessful", log.ERROR, log.Error(err), data)
	http.Error(w, err.Error(), status)
}
