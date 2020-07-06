package api

import (
	"net/http"
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

	// Acquire lock for image ID, and defer unlocking
	lockID, err := api.mongoDB.AcquireImageLock(ctx, id)
	if err != nil {
		handleError(ctx, w, err, logdata)
		return
	}
	defer api.mongoDB.UnlockImage(lockID)

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

// UploadImageHandler updates the image with the provided upload path and sends a kafka 'image uploaded' message
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

	// Acquire lock for image ID, and defer unlocking
	lockID, err := api.mongoDB.AcquireImageLock(ctx, id)
	if err != nil {
		handleError(ctx, w, err, logdata)
		return
	}
	defer api.mongoDB.UnlockImage(lockID)

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

	// Update image in mongo DB
	_, err = api.mongoDB.UpdateImage(ctx, id, imageUpdate)
	if err != nil {
		handleError(ctx, w, err, logdata)
		return
	}

	// Generate the kafka event to trigger the upload
	log.Event(ctx, "sending image uploaded message", log.INFO, logdata)
	event := event.ImageUploaded{
		ImageID: id,
		Path:    upload.Path,
	}
	api.uploadProducer.ImageUploaded(&event)

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

	// Acquire lock for image ID, and defer unlocking
	lockID, err := api.mongoDB.AcquireImageLock(ctx, id)
	if err != nil {
		handleError(ctx, w, err, logdata)
		return
	}
	defer api.mongoDB.UnlockImage(lockID)

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

	// Update image in mongo DB
	_, err = api.mongoDB.UpdateImage(ctx, id, imageUpdate)
	if err != nil {
		handleError(ctx, w, err, logdata)
		return
	}

	// Send 'image published' kafka message
	log.Event(ctx, "sending image published message", log.INFO, logdata)
	event := event.ImagePublished{
		ImageID: id,
	}
	api.publishedProducer.ImagePublished(&event)
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

	// Acquire lock for image ID, and defer unlocking
	lockID, err := api.mongoDB.AcquireImageLock(ctx, id)
	if err != nil {
		handleError(ctx, w, err, logdata)
		return
	}
	defer api.mongoDB.UnlockImage(lockID)

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

	// if the high level image is not already in importing state, validate that the transition is allowed
	if existingImage.State != models.StateImporting.String() && !existingImage.StateTransitionAllowed(models.StateImporting.String()) {
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

// CompleteVariantHandler is a handler that completes a download variant for an image,
// and it also completes the image if all its variants have been completed.
func (api *API) CompleteVariantHandler(w http.ResponseWriter, req *http.Request) {
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

	// complete update
	now := time.Now()
	imageUpdate := &models.Image{
		Downloads: map[string]models.Download{
			variant: {
				State:            models.StateDownloadCompleted.String(),
				PublishCompleted: &now,
			},
		},
	}

	// Acquire lock for image ID, and defer unlocking
	lockID, err := api.mongoDB.AcquireImageLock(ctx, id)
	if err != nil {
		handleError(ctx, w, err, logdata)
		return
	}
	defer api.mongoDB.UnlockImage(lockID)

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

	// The high level image needs to be in 'published' state to allow a complete variant operation
	if existingImage.State != models.StatePublished.String() {
		handleError(ctx, w, apierrors.ErrImageNotPublished, logdata)
		return
	}

	// validate that the download variant state transition is allowed
	if !downloadVariant.StateTransitionAllowed(models.StateDownloadCompleted.String()) {
		logdata["current_state"] = existingImage.State
		logdata["target_state"] = imageUpdate.State
		handleError(ctx, w, apierrors.ErrVariantStateTransitionNotAllowed, logdata)
		return
	}

	// if all downloads are completed, then set the high level state to completed
	if existingImage.AllOtherDownloadsCompleted(variant) {
		imageUpdate.State = models.StateCompleted.String()
	}

	// if any download failed, then set the high level state to failed_publish and if all of them are completed, set it to completed.
	if existingImage.AnyDownloadFailed() {
		log.Event(ctx, "setting high level image state to 'failed_publish' because at least one of the "+
			"other download variants for this image failed to import", log.WARN, logdata)
		imageUpdate.State = models.StateFailedPublish.String()
	} else if existingImage.AllOtherDownloadsCompleted(variant) {
		log.Event(ctx, "setting high level image state to 'completed' because all download variants have been completed", log.INFO, logdata)
		imageUpdate.State = models.StateCompleted.String()
	}

	// Update image in mongo DB
	_, err = api.mongoDB.UpdateImage(ctx, id, imageUpdate)
	if err != nil {
		handleError(ctx, w, err, logdata)
		return
	}

}
