package apierrors

import (
	"errors"
)

// A list of error messages for Image API
var (
	ErrImageNotFound       = errors.New("image not found")
	ErrInternalServer      = errors.New("internal error")
	ErrResourceState       = errors.New("incorrect resource state")
	ErrUnableToReadMessage = errors.New("failed to read message body")
	ErrColIDMismatch       = errors.New("'Collection-Id' header does not match 'collection_id' query parameter")
	ErrUnableToParseJSON   = errors.New("failed to parse json body")
)
