package models

import (
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
	ID           string                         `bson:"_id,omitempty"           json:"id,omitempty"`
	CollectionID string                         `bson:"collection_id,omitempty" json:"collection_id,omitempty"`
	State        string                         `bson:"state,omitempty"         json:"state,omitempty"`
	Filename     string                         `bson:"filename,omitempty"      json:"filename,omitempty"`
	License      *License                       `bson:"license,omitempty"       json:"license,omitempty"`
	Upload       *Upload                        `bson:"upload,omitempty"        json:"upload,omitempty"`
	Type         string                         `bson:"type,omitempty"          json:"type,omitempty"`
	Downloads    map[string]map[string]Download `bson:"downloads,omitempty"     json:"downloads,omitempty"`
}

// License represents a license model
type License struct {
	Title string `bson:"title,omitempty"            json:"title,omitempty"`
	Href  string `bson:"href,omitempty"             json:"href,omitempty"`
}

// Upload represents an upload model
type Upload struct {
	Path string `bson:"path,omitempty"              json:"path,omitempty"`
}

// Download represents a download variant model
type Download struct {
	Size    int    `bson:"size,omitempty"           json:"size,omitempty"`
	Href    string `bson:"href,omitempty"           json:"href,omitempty"`
	Public  string `bson:"public,omitempty"         json:"public,omitempty"`
	Private string `bson:"private,omitempty"        json:"private,omitempty"`
}

// Validate checks that an image struct complies with the filename and state constraints
func (i *Image) Validate() error {

	if len(i.Filename) > MaxFilenameLen {
		return apierrors.ErrImageFilenameTooLong
	}

	if _, err := ParseState(i.State); err != nil {
		return apierrors.ErrImageInvalidState
	}

	return nil
}

// StateTransitionAllowed checks if the image can transition from its current state to the provided target state
func (i *Image) StateTransitionAllowed(target string) bool {
	currentState, err := ParseState(i.State)
	if err != nil {
		return false
	}
	targetState, err := ParseState(target)
	if err != nil {
		return false
	}
	return currentState.TransitionAllowed(targetState)
}
