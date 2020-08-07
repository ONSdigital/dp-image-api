package api

import (
	"context"
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

	// apply the update
	if updatedImage := api.doUpdateImage(w, req, id, image, logdata); updatedImage != nil {
		if err := WriteJSONBody(ctx, updatedImage, w, logdata); err != nil {
			handleError(ctx, w, err, logdata)
			return
		}
		log.Event(ctx, "successfully updated image", log.INFO, logdata)
	}
}

// doUpdateImage is a function to apply the provided image update after doing all the necessary validations, it returns the updated image, or nil if it was not updated
func (api *API) doUpdateImage(w http.ResponseWriter, req *http.Request, id string, image *models.Image, logdata log.Data) (updatedImage *models.Image) {
	ctx := req.Context()

	if err := image.Validate(); err != nil {
		handleError(ctx, w, err, logdata)
		return nil
	}
	log.Event(ctx, "updating image", log.INFO, log.Data{"image": image})

	// Validate a possible mismatch of image id, if provided in image body
	if image.ID != "" && image.ID != id {
		handleError(ctx, w, apierrors.ErrImageIDMismatch, logdata)
		return nil
	}
	image.ID = id

	// Acquire lock for image ID, and defer unlocking
	lockID, err := api.mongoDB.AcquireImageLock(ctx, id)
	if err != nil {
		handleError(ctx, w, err, logdata)
		return nil
	}
	defer api.unlockImage(ctx, lockID)

	// get existing image from mongoDB by id
	existingImage, err := api.mongoDB.GetImage(req.Context(), id)
	if err != nil {
		handleError(ctx, w, err, logdata)
		return nil
	}

	// check that state transition is allowed, only if state is provided
	if image.State != "" {
		if !existingImage.StateTransitionAllowed(image.State) {
			logdata["current_state"] = existingImage.State
			logdata["target_state"] = image.State
			handleError(ctx, w, apierrors.ErrImageStateTransitionNotAllowed, logdata)
			return nil
		}
	}

	// if the image is already published, it cannot be updated
	if existingImage.State == models.StatePublished.String() {
		handleError(ctx, w, apierrors.ErrImageAlreadyPublished, logdata)
		return nil
	}

	// if the image is already completed, it cannot be updated
	if existingImage.State == models.StateCompleted.String() {
		handleError(ctx, w, apierrors.ErrImageAlreadyCompleted, logdata)
		return nil
	}

	// Update image in mongo DB
	didChange, err := api.mongoDB.UpdateImage(ctx, id, image)
	if err != nil {
		handleError(ctx, w, err, logdata)
		return nil
	}

	// get updated image from mongoDB by id (if it changed)
	updatedImage = existingImage
	if didChange {
		updatedImage, err = api.mongoDB.GetImage(req.Context(), id)
		if err != nil {
			handleError(ctx, w, err, logdata)
			return nil
		}
	}
	return updatedImage
}

// CreateDownloadHandler is a handler that adds a new download to an existing image
func (api *API) CreateDownloadHandler(w http.ResponseWriter, req *http.Request) {
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
	newDownload := &models.Download{}
	if err := ReadJSONBody(ctx, req.Body, newDownload, w, logdata); err != nil {
		handleError(ctx, w, err, logdata)
		return
	}

	// generic download validation
	if err := newDownload.Validate(); err != nil {
		handleError(ctx, w, err, logdata)
		return
	}

	variant := newDownload.ID

	// Creat HATEOS links for download variant
	newDownload.Links = api.createLinksForDownload(id, variant)

	// Check provided variant state supplied is correct
	if newDownload.State != models.StateDownloadImporting.String() {
		handleError(ctx, w, apierrors.ErrImageDownloadBadInitialState, logdata)
		return
	}

	// Acquire lock for image ID, and defer unlocking
	lockID, err := api.mongoDB.AcquireImageLock(ctx, id)
	if err != nil {
		handleError(ctx, w, err, logdata)
		return
	}
	defer api.unlockImage(ctx, lockID)

	// get image from mongoDB by id
	image, err := api.mongoDB.GetImage(req.Context(), id)
	if err != nil {
		handleError(ctx, w, err, logdata)
		return
	}

	// The high level image needs to be in 'uploaded' or 'importing' state to allow an update variant operation
	if image.State != models.StateUploaded.String() && image.State != models.StateImporting.String() {
		handleError(ctx, w, apierrors.ErrImageNotImporting, logdata)
		return
	}

	// check for existing variant
	_, found := image.Downloads[variant]
	if found {
		handleError(ctx, w, apierrors.ErrVariantAlreadyExists, logdata)
		return
	}

	// Add new download to existing image and update image state
	if image.Downloads == nil {
		image.Downloads = map[string]models.Download{}
	}
	image.Downloads[variant] = *newDownload
	image.State = models.StateImporting.String()

	// Update image in mongo DB
	err = api.mongoDB.UpsertImage(ctx, id, image)
	if err != nil {
		handleError(ctx, w, err, logdata)
		return
	}

	w.WriteHeader(http.StatusCreated)
	if err := WriteJSONBody(ctx, newDownload, w, logdata); err != nil {
		handleError(ctx, w, err, logdata)
		return
	}
	log.Event(ctx, "successfully created download variant", log.INFO, logdata)
}

func (api *API) createLinksForDownload(id, variant string) *models.DownloadLinks {
	self := api.urlBuilder.BuildImageDownloadURL(id, variant)
	image := api.urlBuilder.BuildImageURL(id)
	return &models.DownloadLinks{
		Self:  self,
		Image: image,
	}
}

// UpdateDownloadHandler is a handler to update an image download variant
func (api *API) UpdateDownloadHandler(w http.ResponseWriter, req *http.Request) {
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

	// Unmarshal download variant from body and validate it
	download := &models.Download{}
	if err := ReadJSONBody(ctx, req.Body, download, w, logdata); err != nil {
		handleError(ctx, w, err, logdata)
		return
	}

	// create image update with the allowed field. Public, state, href, error or timepstamps can't be provided by the caller.
	now := time.Now()
	imageUpdate := &models.Image{
		Downloads: map[string]models.Download{
			variant: {
				Size:            download.Size,
				Palette:         download.Palette,
				Type:            download.Type,
				Width:           download.Width,
				Height:          download.Height,
				Private:         download.Private,
				State:           models.StateDownloadImported.String(),
				ImportCompleted: &now,
			},
		},
	}

	// Acquire lock for image ID, and defer unlocking
	lockID, err := api.mongoDB.AcquireImageLock(ctx, id)
	if err != nil {
		handleError(ctx, w, err, logdata)
		return
	}
	defer api.unlockImage(ctx, lockID)

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

	// The high level image needs to be in 'importing' state to allow an update variant operation
	if existingImage.State != models.StateImporting.String() {
		handleError(ctx, w, apierrors.ErrImageNotImporting, logdata)
		return
	}

	// validate that the download variant state transition is allowed
	if !downloadVariant.StateTransitionAllowed(models.StateImported.String()) {
		logdata["current_state"] = existingImage.State
		logdata["target_state"] = imageUpdate.State
		handleError(ctx, w, apierrors.ErrVariantStateTransitionNotAllowed, logdata)
		return
	}

	// validate that the type matches the existing type
	if downloadVariant.Type != "" && download.Type != "" && downloadVariant.Type != download.Type {
		handleError(ctx, w, apierrors.ErrImageDownloadTypeMismatch, logdata)
		return
	}

	// if any download failed, then set the high level state to failed_import and if all of them are imported, set it to imported.
	if existingImage.AnyDownloadFailed() {
		log.Event(ctx, "setting high level image state to 'failed_import' because at least one of the "+
			"other download variants for this image failed to import", log.WARN, logdata)
		imageUpdate.State = models.StateFailedImport.String()
	} else if existingImage.AllOtherDownloadsImported(variant) {
		log.Event(ctx, "setting high level image state to 'imported' because all download variants have been successfully imported", log.INFO, logdata)
		imageUpdate.State = models.StateImported.String()
	}

	// Update image in mongo DB
	didChange, err := api.mongoDB.UpdateImage(ctx, id, imageUpdate)
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

	// return the updated download variant as a json object in the response body
	if err := WriteJSONBody(ctx, updatedImage.Downloads[variant], w, logdata); err != nil {
		handleError(ctx, w, err, logdata)
		return
	}
	log.Event(ctx, "successfully updated download variant", log.INFO, logdata)
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
	defer api.unlockImage(ctx, lockID)

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
	for _, variant := range image.Downloads {
		imgURL, err := url.Parse(variant.Href)
		if err != nil {
			return nil, err
		}
		events = append(events, ImagePublishedEvent(imgURL.Path))
	}
	return events, nil
}

// unlockImage unlocks the provided image lockID and logs any error with WARN state
func (api *API) unlockImage(ctx context.Context, lockID string) {
	if err := api.mongoDB.UnlockImage(lockID); err != nil {
		log.Event(ctx, "error unlocking mongoDB lock for an image resource", log.WARN, log.Data{"lockID": lockID})
	}
}

/// TODO Remove Deprecated handlers…

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
	defer api.unlockImage(ctx, lockID)

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
	defer api.unlockImage(ctx, lockID)

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
	defer api.unlockImage(ctx, lockID)

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
			"other download variants for this image failed to publish", log.WARN, logdata)
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
