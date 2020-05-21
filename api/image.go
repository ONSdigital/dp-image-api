package api

import (
	"net/http"

	"github.com/ONSdigital/dp-image-api/models"
	"github.com/ONSdigital/log.go/log"
	"github.com/gorilla/mux"
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
func (api *API) GetImagesHandler(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	vars := mux.Vars(req)
	collectionID := vars["collection_id"]
	logdata := log.Data{"request": "get /images", "collectionID": collectionID}

	items, err := api.mongoDB.GetImages(ctx, collectionID)
	if err != nil {
		handleError(ctx, w, err, logdata)
		return
	}

	images := models.Images{
		Items:      items,
		Count:      len(items),
		TotalCount: len(items),
		Limit:      len(items),
	}

	if err := WriteJSONBody(ctx, images, w, logdata); err != nil {
		return
	}
	log.Event(ctx, "Successfully retrieved images for a collection", log.INFO, logdata)
}

// GetImageHandler is a handler that gets an image by its id from MongoDB
func (api *API) GetImageHandler(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	vars := mux.Vars(req)
	id := vars["id"]
	logdata := log.Data{"request": "get /images/{id}", "id": id}

	image, err := api.mongoDB.GetImage(id)
	if err != nil {
		handleError(ctx, w, err, logdata)
		return
	}

	if err := WriteJSONBody(ctx, image, w, logdata); err != nil {
		return
	}
	log.Event(ctx, "Successfully retrieved image", log.INFO, logdata)
}

// UpdateImageHandler is a handler that updates an existing image in MongoDB
func (api *API) UpdateImageHandler(w http.ResponseWriter, req *http.Request) {}

// PublishImageHandler is a handler that triggers the publishing of an image
func (api *API) PublishImageHandler(w http.ResponseWriter, req *http.Request) {}
