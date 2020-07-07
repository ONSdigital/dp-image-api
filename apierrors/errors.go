package apierrors

import (
	"errors"
)

// A list of error messages for Image API
var (
	ErrImageNotFound                    = errors.New("image not found")
	ErrVariantNotFound                  = errors.New("Image download variant not found")
	ErrInternalServer                   = errors.New("internal error")
	ErrUnableToReadMessage              = errors.New("failed to read message body")
	ErrImageIDMismatch                  = errors.New("Image id provided in body does not match 'id' path parameter")
	ErrUnableToParseJSON                = errors.New("failed to parse json body")
	ErrImageFilenameTooLong             = errors.New("image filename is too long")
	ErrImageNoCollectionID              = errors.New("image does not have a collectionID")
	ErrImageAlreadyPublished            = errors.New("image is already published")
	ErrImageAlreadyCompleted            = errors.New("image is already completed")
	ErrImageInvalidState                = errors.New("image state is not a valid state name")
	ErrImageStateTransitionNotAllowed   = errors.New("image state transition not allowed")
	ErrImageNotPublished                = errors.New("image is not in published state")
	ErrVariantStateTransitionNotAllowed = errors.New("image download variant state transition not allowed")
	ErrImageDownloadInvalidState        = errors.New("image download state is not a valid state name")
)
