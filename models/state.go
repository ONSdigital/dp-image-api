package models

import "github.com/ONSdigital/dp-image-api/apierrors"

// State - iota enum of possible image states
type State int

// Possible values for a State of an image. It can only be one of the following:
const (
	StateCreated State = iota
	StateUploaded
	StatePublishing
	StatePublished
	StateDeleted
)

var stateValues = []string{"created", "uploaded", "publishing", "published", "deleted"}

// String returns the string representation of a state
func (s State) String() string {
	return stateValues[s]
}

// ParseState returns a state from its string representation
func ParseState(stateStr string) (State, error) {
	for s, validState := range stateValues {
		if stateStr == validState {
			return State(s), nil
		}
	}
	return -1, apierrors.ErrImageInvalidState
}

// TransitionAllowed returns true only if the transition from the current state and the provided targetState is allowed
func (s State) TransitionAllowed(target State) bool {
	switch s {
	case StateCreated:
		switch target {
		case StateCreated, StateUploaded, StateDeleted:
			return true
		default:
			return false
		}
	case StateUploaded:
		switch target {
		case StateUploaded, StatePublishing, StateDeleted:
			return true
		default:
			return false
		}
	case StatePublishing:
		switch target {
		case StatePublishing, StatePublished, StateUploaded, StateDeleted:
			return true
		default:
			return false
		}
	case StatePublished:
		switch target {
		case StatePublished, StateDeleted:
			return true
		default:
			return false
		}
	default:
		return false
	}
}
