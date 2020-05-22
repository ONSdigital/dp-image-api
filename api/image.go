package api

import (
	"net/http"

	"github.com/ONSdigital/dp-image-api/apierrors"
	"github.com/ONSdigital/dp-image-api/models"
	"github.com/ONSdigital/dp-net/handlers"
	dpHTTP "github.com/ONSdigital/dp-net/http"
	"github.com/ONSdigital/log.go/log"
	"github.com/gorilla/mux"
	uuid "github.com/satori/go.uuid"
)

// CreateImageHandler is a handler that upserts an image into mongoDB
func (api *API) CreateImageHandler(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	logdata := log.Data{
		handlers.CollectionID.Header(): ctx.Value(handlers.CollectionID.Context()),
		"request-id":                   ctx.Value(dpHTTP.RequestIdKey),
	}

	newImageRequest := &models.Image{}
	if err := ReadJSONBody(ctx, req.Body, newImageRequest, w, logdata); err != nil {
		return
	}

	// generate new image from request, mapping only allowed fields at creation time (model newImage in swagger spec)
	newImage := models.Image{
		ID:           uuid.NewV4().String(),
		CollectionID: newImageRequest.CollectionID,
		State:        models.StateCreated.String(),
		Filename:     newImageRequest.Filename,
		License:      newImageRequest.License,
		Type:         newImageRequest.Type,
	}
	log.Event(ctx, "storing new image", log.INFO, log.Data{"image": newImage})

	if err := api.mongoDB.UpsertImage(newImage.ID, &newImage); err != nil {
		handleError(ctx, w, err, logdata)
		return
	}

	w.WriteHeader(http.StatusCreated)
	if err := WriteJSONBody(ctx, newImage, w, logdata); err != nil {
		return
	}
	log.Event(ctx, "successfully created image", log.INFO, logdata)
}

// GetImagesHandler is a handler that gets all images in a collection from MongoDB
func (api *API) GetImagesHandler(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	hColID := ctx.Value(handlers.CollectionID.Context())
	logdata := log.Data{
		handlers.CollectionID.Header(): hColID,
		"request-id":                   ctx.Value(dpHTTP.RequestIdKey),
	}

	// validate collection ID from header matches collection ID from query param
	colID := req.URL.Query().Get("collection_id")
	if colID != hColID {
		handleError(ctx, w, apierrors.ErrColIDMismatch, logdata)
		return
	}

	// get images from MongoDB for the requested collection
	items, err := api.mongoDB.GetImages(ctx, colID)
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
	hColID := ctx.Value(handlers.CollectionID.Context())
	logdata := log.Data{
		handlers.CollectionID.Header(): hColID,
		"request-id":                   ctx.Value(dpHTTP.RequestIdKey),
		"image-id":                     id,
	}

	// get image from mongoDB by id
	image, err := api.mongoDB.GetImage(id)
	if err != nil {
		handleError(ctx, w, err, logdata)
		return
	}

	// if image is not published, vaidate that its collectionID matches que header collection-Id
	if image.State != models.StatePublished.String() && image.CollectionID != hColID {
		handleError(ctx, w, apierrors.ErrColIDMismatch, logdata)
		return
	}

	if err := WriteJSONBody(ctx, image, w, logdata); err != nil {
		return
	}
	log.Event(ctx, "Successfully retrieved image", log.INFO, logdata)
}

// UpdateImageHandler is a handler that updates an existing image in MongoDB
func (api *API) UpdateImageHandler(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	logdata := log.Data{
		handlers.CollectionID.Header(): ctx.Value(handlers.CollectionID.Context()),
		"request-id":                   ctx.Value(dpHTTP.RequestIdKey),
	}
	log.Event(ctx, "update image was called, but it is not implemented yet", log.INFO, logdata)
	http.Error(w, "Not implemented", http.StatusNotImplemented)
}

// PublishImageHandler is a handler that triggers the publishing of an image
func (api *API) PublishImageHandler(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	logdata := log.Data{
		handlers.CollectionID.Header(): ctx.Value(handlers.CollectionID.Context()),
		"request-id":                   ctx.Value(dpHTTP.RequestIdKey),
	}
	log.Event(ctx, "update image was called, but it is not implemented yet", log.INFO, logdata)
	http.Error(w, "Not implemented", http.StatusNotImplemented)
}
