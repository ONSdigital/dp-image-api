package api

import (
	"net/http"
	"net/url"
	"time"

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

	// refresh to populate any fields that are not stored in mongoDB
	images.Refresh()

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

	// refresh to populate any fields that are not stored in mongoDB
	image.Refresh()

	if err := WriteJSONBody(ctx, image, w, logdata); err != nil {
		handleError(ctx, w, err, logdata)
		return
	}
	log.Event(ctx, "Successfully retrieved image", log.INFO, logdata)
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

	// PUT should not affect the state so clear it if it is supplied
	image.State = ""

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

	// TODO lock with defer unlock

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

	// if the image is already completed, it cannot be updated
	if existingImage.State == models.StateCompleted.String() {
		handleError(ctx, w, apierrors.ErrImageAlreadyCompleted, logdata)
		return
	}

	// Update image in mongo DB
	didChange, err := api.mongoDB.UpdateImage(ctx, id, image)
	if err != nil {
		handleError(ctx, w, err, logdata)
		return
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

func (api *API) UploadImageHandler(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	vars := mux.Vars(req)
	id := vars["id"]
	hColID := ctx.Value(handlers.CollectionID.Context())
	logdata := log.Data{
		handlers.CollectionID.Header(): hColID,
		"request-id":                   ctx.Value(dphttp.RequestIdKey),
		"image-id":                     id,
	}

	// Unmarshal upload from body and validate it
	upload := &models.Upload{}
	if err := ReadJSONBody(ctx, req.Body, upload, w, logdata); err != nil {
		handleError(ctx, w, err, logdata)
		return
	}

	imageUpdate := &models.Image{
		State:  models.StateUploaded.String(),
		Upload: upload,
	}

	// get existing image from mongoDB by id
	existingImage, err := api.mongoDB.GetImage(req.Context(), id)
	if err != nil {
		handleError(ctx, w, err, logdata)
		return
	}

	// validate that the uploaded transition state is allowed
	if !existingImage.StateTransitionAllowed(imageUpdate.State) {
		logdata["current_state"] = existingImage.State
		logdata["target_state"] = imageUpdate.State
		handleError(ctx, w, apierrors.ErrImageStateTransitionNotAllowed, logdata)
		return
	}

	// Generate and send the kafka event to trigger the upload
	log.Event(ctx, "sending image uploaded message", log.INFO, logdata)
	event := ImageUploadedEvent(id, upload.Path)
	if err := api.uploadProducer.ImageUploaded(event); err != nil {
		handleError(ctx, w, err, logdata)
		return
	}

	// Update image in mongo DB
	_, err = api.mongoDB.UpdateImage(ctx, id, imageUpdate)
	if err != nil {
		handleError(ctx, w, err, logdata)
		return
	}
}

// PublishImageHandler is a handler that triggers the publishing of an image
func (api *API) PublishImageHandler(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	vars := mux.Vars(req)
	id := vars["id"]
	hColID := ctx.Value(handlers.CollectionID.Context())
	logdata := log.Data{
		handlers.CollectionID.Header(): hColID,
		"request-id":                   ctx.Value(dphttp.RequestIdKey),
		"image-id":                     id,
	}

	imageUpdate := &models.Image{State: models.StatePublished.String()}

	// get image from mongoDB by id
	existingImage, err := api.mongoDB.GetImage(req.Context(), id)
	if err != nil {
		handleError(ctx, w, err, logdata)
		return
	}

	// validate that the publish transition state is allowed
	if !existingImage.StateTransitionAllowed(imageUpdate.State) {
		logdata["current_state"] = existingImage.State
		logdata["target_state"] = imageUpdate.State
		handleError(ctx, w, apierrors.ErrImageStateTransitionNotAllowed, logdata)
		return
	}

	// Generate 'image published' events for all download variants
	events, err := generateImagePublishEvents(existingImage)
	if err != nil {
		handleError(ctx, w, err, logdata)
		return
	}

	// Send 'image published' kafka messages corresponding to all the download variants
	log.Event(ctx, "sending image published messages", log.INFO, logdata)
	for _, e := range events {
		if err := api.publishedProducer.ImagePublished(e); err != nil {
			handleError(ctx, w, err, logdata)
			return
		}
	}

	// Update image in mongo DB
	_, err = api.mongoDB.UpdateImage(ctx, id, imageUpdate)
	if err != nil {
		handleError(ctx, w, err, logdata)
		return
	}
}

// generateImagePublishEvents creates a kafka 'image-published' event for each download variant for the provided image.
// Note that the private and public paths will be the same, according to the way the URLs are constructed in 'Refresh()' method,
// using DownloadHrefFmt format "http://<host>/images/<imageID>/<variantName>/<fileName>"
func generateImagePublishEvents(image *models.Image) (events []*event.ImagePublished, err error) {
	image.Refresh()
	for _, variant := range image.Downloads {
		imgURL, err := url.Parse(variant.Href)
		if err != nil {
			return nil, err
		}
		events = append(events, ImagePublishedEvent(imgURL.Path))
	}
	return events, nil
}

//ImageUploadedEvent returns an ImageUploaded event for the provided image ID and upload path
var ImageUploadedEvent = func(imageID, uploadPath string) *event.ImageUploaded {
	return &event.ImageUploaded{
		ImageID: imageID,
		Path:    uploadPath,
	}
}

// ImagePublishedEvent returns an ImagePublished event for the provided path
var ImagePublishedEvent = func(path string) *event.ImagePublished {
	return &event.ImagePublished{
		SrcPath: path,
		DstPath: path,
	}
}

// ImportVariantHandler is a handler that imports a download variant for an image
func (api *API) ImportVariantHandler(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	vars := mux.Vars(req)
	id := vars["id"]
	variant := vars["variant"]
	hColID := ctx.Value(handlers.CollectionID.Context())
	logdata := log.Data{
		handlers.CollectionID.Header(): hColID,
		"request-id":                   ctx.Value(dphttp.RequestIdKey),
		"image-id":                     id,
		"download-variant":             variant,
	}

	// image import to perform
	now := time.Now()
	imageUpdate := &models.Image{
		State: models.StateImporting.String(),
		Downloads: map[string]models.Download{
			variant: {
				State:         models.StateDownloadImporting.String(),
				ImportStarted: &now,
			},
		},
	}

	// TODO lock mongoDB with defer unlock

	// get image from mongoDB by id
	existingImage, err := api.mongoDB.GetImage(req.Context(), id)
	if err != nil {
		handleError(ctx, w, err, logdata)
		return
	}

	// get requested variant
	downloadVariant, found := existingImage.Downloads[variant]
	if !found {
		handleError(ctx, w, apierrors.ErrVariantNotFound, logdata)
		return
	}

	// validate that the high level importing transition state is allowed
	if !existingImage.StateTransitionAllowed(models.StateImporting.String()) {
		logdata["current_state"] = existingImage.State
		logdata["target_state"] = imageUpdate.State
		handleError(ctx, w, apierrors.ErrImageStateTransitionNotAllowed, logdata)
		return
	}

	// validate that the download variant state transition is allowed
	if !downloadVariant.StateTransitionAllowed(models.StateDownloadImporting.String()) {
		logdata["current_state"] = existingImage.State
		logdata["target_state"] = imageUpdate.State
		handleError(ctx, w, apierrors.ErrVariantStateTransitionNotAllowed, logdata)
		return
	}

	// Update image in mongo DB
	_, err = api.mongoDB.UpdateImage(ctx, id, imageUpdate)
	if err != nil {
		handleError(ctx, w, err, logdata)
		return
	}
}
