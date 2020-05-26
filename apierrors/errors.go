package apierrors

import (
	"errors"
)

// A list of error messages for Image API
var (
	ErrImageNotFound        = errors.New("image not found")
	ErrInternalServer       = errors.New("internal error")
	ErrResourceState        = errors.New("incorrect resource state")
	ErrUnableToReadMessage  = errors.New("failed to read message body")
	ErrColIDMismatch        = errors.New("'Collection-Id' header does not match 'collection_id' query parameter")
	ErrWrongColID           = errors.New("'Collection-Id' header does not match image's 'collection_id'")
	ErrUnableToParseJSON    = errors.New("failed to parse json body")
	ErrImageFilenameTooLong = errors.New("image filename is too long")
	ErrImageInvalidState    = errors.New("image state is not a valid state name")
)
