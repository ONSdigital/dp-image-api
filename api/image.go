package api

import (
	"net/http"

	"github.com/ONSdigital/dp-image-api/apierrors"
	"github.com/ONSdigital/dp-image-api/event"
	"github.com/ONSdigital/dp-image-api/models"
	"github.com/ONSdigital/dp-net/handlers"
	dphttp "github.com/ONSdigital/dp-net/http"
	"github.com/ONSdigital/log.go/log"
	"github.com/gorilla/mux"
	uuid "github.com/satori/go.uuid"
)

// NewID returns a new UUID
var NewID = func() string {
	return uuid.NewV4().String()
}

// CreateImageHandler is a handler that inserts an image into mongoDB with a newly generated ID
func (api *API) CreateImageHandler(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	hColID := ctx.Value(handlers.CollectionID.Context())
	logdata := log.Data{
		handlers.CollectionID.Header(): hColID,
		"request-id":                   ctx.Value(dphttp.RequestIdKey),
	}

	newImageRequest := &models.Image{}
	if err := ReadJSONBody(ctx, req.Body, newImageRequest, w, logdata); err != nil {
		handleError(ctx, w, err, logdata)
		return
	}

	// generate new image from request, mapping only allowed fields at creation time (model newImage in swagger spec)
	// the image is always created in 'created' state, and it is assigned a newly generated ID
	newImage := models.Image{
		ID:           NewID(),
		CollectionID: newImageRequest.CollectionID,
		State:        models.StateCreated.String(),
		Filename:     newImageRequest.Filename,
		License:      newImageRequest.License,
		Type:         newImageRequest.Type,
	}

	// generic image validation
	if err := newImage.Validate(); err != nil {
		handleError(ctx, w, err, logdata)
		return
	}

	// check that collectionID is provided in the request body
	if newImage.CollectionID == "" {
		handleError(ctx, w, apierrors.ErrImageNoCollectionID, logdata)
		return
	}

	log.Event(ctx, "storing new image", log.INFO, log.Data{"image": newImage})

	// Upsert image in MongoDB
	if err := api.mongoDB.UpsertImage(req.Context(), newImage.ID, &newImage); err != nil {
		handleError(ctx, w, err, logdata)
		return
	}

	w.WriteHeader(http.StatusCreated)
	if err := WriteJSONBody(ctx, newImage, w, logdata); err != nil {
		handleError(ctx, w, err, logdata)
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
		"request-id":                   ctx.Value(dphttp.RequestIdKey),
	}

	// get collection_id query parameter (optional)
	colID := req.URL.Query().Get("collection_id")
	logdata["collection_id"] = colID

	// get images from MongoDB with the requested collection_id filter, if provided.
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
		handleError(ctx, w, err, logdata)
		return
	}
	log.Event(ctx, "Successfully retrieved images", log.INFO, logdata)
}

// GetImageHandler is a handler that gets an image by its id from MongoDB
func (api *API) GetImageHandler(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	vars := mux.Vars(req)
	id := vars["id"]
	hColID := ctx.Value(handlers.CollectionID.Context())
	logdata := log.Data{
		handlers.CollectionID.Header(): hColID,
		"request-id":                   ctx.Value(dphttp.RequestIdKey),
		"image-id":                     id,
	}

	// get image from mongoDB by id
	image, err := api.mongoDB.GetImage(req.Context(), id)
	if err != nil {
		handleError(ctx, w, err, logdata)
		return
	}

	if err := WriteJSONBody(ctx, image, w, logdata); err != nil {
		handleError(ctx, w, err, logdata)
		return
	}
	log.Event(ctx, "Successfully retrieved image", log.INFO, logdata)
}

// isUploadOperation returns true if the provided imageUpdate defines an upload operation (i.e. Upload path provided)
func isUploadOperation(imageUpdate *models.Image) bool {
	return imageUpdate.Upload != nil && imageUpdate.Upload.Path != ""
}

// UpdateImageHandler is a handler that updates an existing image in MongoDB
func (api *API) UpdateImageHandler(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	vars := mux.Vars(req)
	id := vars["id"]
	hColID := ctx.Value(handlers.CollectionID.Context())
	logdata := log.Data{
		handlers.CollectionID.Header(): hColID,
		"request-id":                   ctx.Value(dphttp.RequestIdKey),
		"image-id":                     id,
	}

	// Unmarshal image from body and validate it
	image := &models.Image{}
	if err := ReadJSONBody(ctx, req.Body, image, w, logdata); err != nil {
		handleError(ctx, w, err, logdata)
		return
	}
	if err := image.Validate(); err != nil {
		handleError(ctx, w, err, logdata)
		return
	}
	log.Event(ctx, "updating image", log.INFO, log.Data{"image": image})

	// Validate a possible mismatch of image id, if provided in image body
	if image.ID != "" && image.ID != id {
		handleError(ctx, w, apierrors.ErrImageIDMismatch, logdata)
		return
	}
	image.ID = id

	// get existing image from mongoDB by id
	existingImage, err := api.mongoDB.GetImage(req.Context(), id)
	if err != nil {
		handleError(ctx, w, err, logdata)
		return
	}

	// check that state transition is allowed, only if state is provided
	if image.State != "" {
		if !existingImage.StateTransitionAllowed(image.State) {
			logdata["current_state"] = existingImage.State
			logdata["target_state"] = image.State
			handleError(ctx, w, apierrors.ErrImageStateTransitionNotAllowed, logdata)
			return
		}
	}

	// if the image is already published, it cannot be updated
	if existingImage.State == models.StatePublished.String() {
		handleError(ctx, w, apierrors.ErrImageAlreadyPublished, logdata)
		return
	}

	// Update image in mongo DB
	didChange, err := api.mongoDB.UpdateImage(ctx, id, image)
	if err != nil {
		handleError(ctx, w, err, logdata)
		return
	}

	// If it is an upload operation, generate the kafka event to trigger it
	if isUploadOperation(image) {
		log.Event(ctx, "sending image uploaded message", log.INFO, logdata)
		event := event.ImageUploaded{
			ImageID: image.ID,
			Path:    image.Upload.Path,
		}
		api.producer.ImageUploaded(&event)
	}

	// get updated image from mongoDB by id (if it changed)
	updatedImage := existingImage
	if didChange {
		updatedImage, err = api.mongoDB.GetImage(req.Context(), id)
		if err != nil {
			handleError(ctx, w, err, logdata)
			return
		}
	}

	if err := WriteJSONBody(ctx, updatedImage, w, logdata); err != nil {
		handleError(ctx, w, err, logdata)
		return
	}
	log.Event(ctx, "successfully updated image", log.INFO, logdata)

}

// PublishImageHandler is a handler that triggers the publishing of an image
func (api *API) PublishImageHandler(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	logdata := log.Data{
		handlers.CollectionID.Header(): ctx.Value(handlers.CollectionID.Context()),
		"request-id":                   ctx.Value(dphttp.RequestIdKey),
	}
	log.Event(ctx, "publish image was called, but it is not implemented yet", log.INFO, logdata)
	http.Error(w, "Not implemented", http.StatusNotImplemented)
}
