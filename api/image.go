package api

import (
	"net/http"

	"github.com/ONSdigital/dp-image-api/models"
	"github.com/ONSdigital/log.go/log"
)

// CreateImageHandler is a handler that upserts an image into mongoDB
func (api *API) CreateImageHandler(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	logdata := log.Data{"request": "post /images"}

	newImage := models.Image{}
	if err := ReadJSONBody(ctx, req.Body, &newImage, w, logdata); err != nil {
		return
	}

	// TODO create image in MongoDB

	w.WriteHeader(http.StatusCreated)
	if err := WriteJSONBody(ctx, newImage, w, logdata); err != nil {
		return
	}
	log.Event(ctx, "(noop) successfully created image", log.INFO, logdata)
}

// GetImagesHandler is a handler that gets all images in a collection from MongoDB
func (api *API) GetImagesHandler(w http.ResponseWriter, req *http.Request) {}

// GetImageHandler is a handler that gets an image by its id from MongoDB
func (api *API) GetImageHandler(w http.ResponseWriter, req *http.Request) {}

// UpdateImageHandler is a handler that updates an existing image in MongoDB
func (api *API) UpdateImageHandler(w http.ResponseWriter, req *http.Request) {}

// PublishImageHandler is a handler that triggers the publishing of an image
func (api *API) PublishImageHandler(w http.ResponseWriter, req *http.Request) {}
