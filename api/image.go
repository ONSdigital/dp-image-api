package api

import (
	"context"
	"net/http"
	"net/url"
	"path"

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
var ImageUploadedEvent = func(imageID, uploadPath, filename string) *event.ImageUploaded {
	return &event.ImageUploaded{
		ImageID:  imageID,
		Path:     uploadPath,
		Filename: filename,
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

	id := NewID()

	// generate new image from request, mapping only allowed fields at creation time (model newImage in swagger spec)
	// the image is always created in 'created' state, and it is assigned a newly generated ID
	newImage := models.Image{
		ID:           id,
		CollectionID: newImageRequest.CollectionID,
		State:        newImageRequest.State,
		Filename:     newImageRequest.Filename,
		License:      newImageRequest.License,
		Links:        api.createLinksForImage(id),
		Type:         newImageRequest.Type,
	}

	// generic image validation
	if err := newImage.Validate(); err != nil {
		handleError(ctx, w, err, logdata)
		return
	}

	// Check provided image state supplied is correct
	if newImage.State != models.StateCreated.String() {
		handleError(ctx, w, apierrors.ErrImageBadInitialState, logdata)
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

	// Validate new model regardless of existing state
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

	// Check that transition from the existing image state is valid
	if err := image.ValidateTransitionFrom(existingImage); err != nil {
		logdata["current_image_state"] = existingImage.State
		logdata["target_image_state"] = image.State
		handleError(ctx, w, err, logdata)
		return nil
	}

	// Copy existing Links to newly updated image
	image.Links = existingImage.Links

	// If the new state is 'uploaded', generate and send the kafka event to trigger import
	if image.State == models.StateUploaded.String() {
		uploadS3Path := path.Base(image.Upload.Path)
		log.Event(ctx, "sending image uploaded message", log.INFO, logdata)
		event := ImageUploadedEvent(id, uploadS3Path, image.Filename)
		if err := api.uploadProducer.ImageUploaded(event); err != nil {
			handleError(ctx, w, err, logdata)
			return
		}
	}

	// Update image in mongo DB
	err = api.mongoDB.UpsertImage(ctx, id, image)
	if err != nil {
		handleError(ctx, w, err, logdata)
		return
	}
	return image
}

// GetDownloadsHandler is a handler that returns all the download variant for an image
func (api *API) GetDownloadsHandler(w http.ResponseWriter, req *http.Request) {
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

	downloadsList := []models.Download{}
	for _, dl := range image.Downloads {
		downloadsList = append(downloadsList, dl)
	}
	downloads := models.Downloads{
		Items:      downloadsList,
		Count:      len(downloadsList),
		TotalCount: len(downloadsList),
		Limit:      len(downloadsList),
	}

	if err := WriteJSONBody(ctx, downloads, w, logdata); err != nil {
		handleError(ctx, w, err, logdata)
		return
	}
	log.Event(ctx, "Successfully retrieved downloads", log.INFO, logdata)
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

	// Create HATEOS links for download variant
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

	// Validate new download against parent image
	if err := newDownload.ValidateForImage(image); err != nil {
		handleError(ctx, w, err, logdata)
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

func (api *API) createLinksForImage(id string) *models.ImageLinks {
	self := api.urlBuilder.BuildImageURL(id)
	downloads := api.urlBuilder.BuildImageDownloadsURL(id)
	return &models.ImageLinks{
		Self:      self,
		Downloads: downloads,
	}
}

func (api *API) createLinksForDownload(id, variant string) *models.DownloadLinks {
	self := api.urlBuilder.BuildImageDownloadURL(id, variant)
	image := api.urlBuilder.BuildImageURL(id)
	return &models.DownloadLinks{
		Self:  self,
		Image: image,
	}
}

// GetDownloadHandler is a handler that returns an individual download variant
func (api *API) GetDownloadHandler(w http.ResponseWriter, req *http.Request) {
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

	// get image from mongoDB by id
	image, err := api.mongoDB.GetImage(req.Context(), id)
	if err != nil {
		handleError(ctx, w, err, logdata)
		return
	}

	download, found := image.Downloads[variant]
	if !found {
		handleError(ctx, w, apierrors.ErrVariantNotFound, logdata)
		return
	}

	if err := WriteJSONBody(ctx, download, w, logdata); err != nil {
		handleError(ctx, w, err, logdata)
		return
	}
	log.Event(ctx, "Successfully retrieved download", log.INFO, logdata)
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

	// Validate a possible mismatch of variant id
	if download.ID != "" && download.ID != variant {
		handleError(ctx, w, apierrors.ErrVariantIDMismatch, logdata)
		return
	}
	download.ID = variant

	// generic download validation
	if err := download.Validate(); err != nil {
		handleError(ctx, w, err, logdata)
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

	// check for existing variant
	existing, found := image.Downloads[variant]
	if !found {
		handleError(ctx, w, apierrors.ErrVariantNotFound, logdata)
		return
	}

	// Validate download variant against parent image state
	if err := download.ValidateForImage(image); err != nil {
		logdata["current_image_state"] = image.State
		logdata["target_download_state"] = download.State
		handleError(ctx, w, err, logdata)
		return
	}

	// Validate download variant against existing variant state
	if err := download.ValidateTransitionFrom(&existing); err != nil {
		logdata["current_image_state"] = existing.State
		logdata["target_download_state"] = download.State
		handleError(ctx, w, err, logdata)
		return
	}

	//Copy Links from existing
	download.Links = existing.Links

	// Update new download to existing image
	image.Downloads[variant] = *download

	// Update image state based on change to download
	image.State = image.UpdatedState()

	// Update image in mongo DB
	err = api.mongoDB.UpsertImage(ctx, id, image)
	if err != nil {
		handleError(ctx, w, err, logdata)
		return
	}

	if err := WriteJSONBody(ctx, download, w, logdata); err != nil {
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
		logdata["current_image_state"] = existingImage.State
		logdata["target_image_state"] = imageUpdate.State
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
