package models

import (
	"time"

	"github.com/ONSdigital/dp-image-api/apierrors"
)

// MaxFilenameLen is the maximum number of characters allowed for Image filenames
const MaxFilenameLen = 40

// Images represents an array of images model as it is stored in mongoDB and json representation for API
type Images struct {
	Count      int     `bson:"count,omitempty"        json:"count"`
	Offset     int     `bson:"offset_index,omitempty" json:"offset_index"`
	Limit      int     `bson:"limit,omitempty"        json:"limit"`
	Items      []Image `bson:"items,omitempty"        json:"items"`
	TotalCount int     `bson:"total_count,omitempty"  json:"total_count"`
}

// Image represents an image metadata model as it is stored in mongoDB and json representation for API
type Image struct {
	ID           string              `bson:"_id,omitempty"           json:"id,omitempty"`
	CollectionID string              `bson:"collection_id,omitempty" json:"collection_id,omitempty"`
	State        string              `bson:"state,omitempty"         json:"state,omitempty"`
	Filename     string              `bson:"filename,omitempty"      json:"filename,omitempty"`
	License      *License            `bson:"license,omitempty"       json:"license,omitempty"`
	Links        *ImageLinks         `bson:"links,omitempty"         json:"links,omitempty"`
	Upload       *Upload             `bson:"upload,omitempty"        json:"upload,omitempty"`
	Type         string              `bson:"type,omitempty"          json:"type,omitempty"`
	Downloads    map[string]Download `bson:"downloads,omitempty"     json:"-"`
}

// License represents a license model
type License struct {
	Title string `bson:"title,omitempty"            json:"title,omitempty"`
	Href  string `bson:"href,omitempty"             json:"href,omitempty"`
}

type ImageLinks struct {
	Self      string `bson:"self,omitempty"         json:"self,omitempty"`
	Downloads string `bson:"downloads,omitempty"    json:"downloads,omitempty"`
}

// Upload represents an upload model
type Upload struct {
	Path string `bson:"path,omitempty"              json:"path,omitempty"`
}

// Downloads represents an array of downloads model as it is stored in mongoDB and json representation for API
type Downloads struct {
	Count      int        `bson:"count,omitempty"        json:"count"`
	Offset     int        `bson:"offset_index,omitempty" json:"offset_index"`
	Limit      int        `bson:"limit,omitempty"        json:"limit"`
	Items      []Download `bson:"items,omitempty"        json:"items"`
	TotalCount int        `bson:"total_count,omitempty"  json:"total_count"`
}

// Download represents a download variant model
type Download struct {
	ID               string         `bson:"id,omitempty"                 json:"id,omitempty"`
	Size             *int           `bson:"size,omitempty"               json:"size,omitempty"`
	Palette          string         `bson:"palette,omitempty"            json:"palette,omitempty"`
	Type             string         `bson:"type,omitempty"               json:"type,omitempty"`
	Width            *int           `bson:"width,omitempty"              json:"width,omitempty"`
	Height           *int           `bson:"height,omitempty"             json:"height,omitempty"`
	Public           bool           `json:"public,omitempty"`
	Href             string         `json:"href,omitempty"`
	Links            *DownloadLinks `bson:"links,omitempty"              json:"links,omitempty"`
	Private          string         `bson:"private,omitempty"            json:"private,omitempty"`
	State            string         `bson:"state,omitempty"              json:"state,omitempty"`
	Error            string         `bson:"error,omitempty"              json:"error,omitempty"`
	ImportStarted    *time.Time     `bson:"import_started,omitempty"     json:"import_started,omitempty"`
	ImportCompleted  *time.Time     `bson:"import_completed,omitempty"   json:"import_completed,omitempty"`
	PublishStarted   *time.Time     `bson:"publish_started,omitempty"    json:"publish_started,omitempty"`
	PublishCompleted *time.Time     `bson:"publish_completed,omitempty"  json:"publish_completed,omitempty"`
}

type DownloadLinks struct {
	Self  string `bson:"self,omitempty"       json:"self,omitempty"`
	Image string `bson:"image,omitempty"      json:"image,omitempty"`
}

// Validate checks that an image struct complies with the filename and state constraints, if provided.
func (i *Image) Validate() error {

	if i.Filename != "" {
		if len(i.Filename) > MaxFilenameLen {
			return apierrors.ErrImageFilenameTooLong
		}
	}

	if i.State != "" {
		if _, err := ParseState(i.State); err != nil {
			return apierrors.ErrImageInvalidState
		}
	}

	// Check uploaded images have a valid upload path
	if i.State == StateUploaded.String() {
		err := validateUpload(i.Upload)
		if err != nil {
			return err
		}
	}

	return nil
}

func validateUpload(upload *Upload) error {
	if upload == nil {
		return apierrors.ErrImageUploadEmpty
	}
	if len(upload.Path) < 1 {
		return apierrors.ErrImageUploadPathEmpty
	}
	return nil
}

// ValidateTransitionFrom checks that this image state can be validly transitioned from the existing state
func (i *Image) ValidateTransitionFrom(existing *Image) error {

	// check that state transition is allowed, only if state is provided
	if i.State != "" {
		if !existing.StateTransitionAllowed(i.State) {
			return apierrors.ErrImageStateTransitionNotAllowed
		}
	}

	// if the image is already completed, it cannot be updated
	if existing.State == StateCompleted.String() {
		return apierrors.ErrImageAlreadyCompleted
	}

	return nil
}

// StateTransitionAllowed checks if the image can transition from its current state to the provided target state
func (i *Image) StateTransitionAllowed(target string) bool {
	currentState, err := ParseState(i.State)
	if err != nil {
		currentState = StateCreated // default value, if state is not present or invalid value
	}
	targetState, err := ParseState(target)
	if err != nil {
		return false
	}
	return currentState.TransitionAllowed(targetState)
}

// UpdatedState returns a new image state based on the image's downloads and existing image state
func (i *Image) UpdatedState() string {

	// Return unchanged state if already failed
	if i.State == StateFailedImport.String() || i.State == StateFailedPublish.String() {
		return i.State
	}

	switch i.State {
	case StateImporting.String():
		if i.AnyDownloadsOfState(StateDownloadFailed) {
			return StateFailedImport.String()
		}
		if i.AllDownloadsOfState(StateDownloadImported) {
			return StateImported.String()
		}
	case StatePublished.String():
		if i.AnyDownloadsOfState(StateDownloadFailed) {
			return StateFailedPublish.String()
		}
		if i.AllDownloadsOfState(StateDownloadCompleted) {
			return StateCompleted.String()
		}
	}

	// Default to existing state
	return i.State
}

// AllDownloadsOfState returns trueOfS if all download variants are in specified state,
func (i *Image) AllDownloadsOfState(s DownloadState) bool {
	if len(i.Downloads) == 0 {
		return false
	}
	for _, download := range i.Downloads {
		if download.State != s.String() {
			return false
		}
	}
	return true
}

// AnyDownloadsOfState returns true if any image download variant is in specified state
func (i *Image) AnyDownloadsOfState(s DownloadState) bool {
	for _, download := range i.Downloads {
		if download.State == s.String() {
			return true
		}
	}
	return false
}

// Validate checks that an download struct complies with the state name constraint, if provided.
func (d *Download) Validate() error {
	if d.State != "" {
		if _, err := ParseDownloadState(d.State); err != nil {
			return apierrors.ErrImageDownloadInvalidState
		}
	}
	return nil
}

// ValidateTransitionFrom checks whether the new state is valid given the existing download state…
func (d *Download) ValidateTransitionFrom(ed *Download) error {

	// validate that the download variant state transition is allowed
	if !ed.StateTransitionAllowed(d.State) {
		return apierrors.ErrVariantStateTransitionNotAllowed
	}

	// validate that the type matches the existing type
	if d.Type != "" && ed.Type != "" && d.Type != ed.Type {
		return apierrors.ErrImageDownloadTypeMismatch
	}

	return nil
}

// ValidateForImage checks whether the new download state is valid for the specified parent image
func (d *Download) ValidateForImage(i *Image) error {

	switch d.State {
	case StateDownloadImporting.String():
		if i.State != StateUploaded.String() && i.State != StateImporting.String() {
			return apierrors.ErrImageNotImporting
		}
	case StateDownloadImported.String():
		if i.State != StateImporting.String() {
			return apierrors.ErrImageNotImporting
		}
	case StateDownloadCompleted.String():
		if i.State == StateCompleted.String() {
			return apierrors.ErrImageAlreadyCompleted
		}
		if i.State != StatePublished.String() {
			return apierrors.ErrImageNotPublished
		}
	}

	return nil
}

// StateTransitionAllowed checks if the download variant can transition from its current state to the provided target state
func (d *Download) StateTransitionAllowed(target string) bool {
	currentState, err := ParseDownloadState(d.State)
	if err != nil {
		currentState = StateDownloadPending // default value, if state is not present or invalid value
	}
	targetState, err := ParseDownloadState(target)
	if err != nil {
		return false
	}
	return currentState.TransitionAllowed(targetState)
}
