package models

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
		case StatePublishing, StateDeleted:
			return true
		default:
			return false
		}
	case StatePublishing:
		switch target {
		case StatePublished, StateUploaded, StateDeleted:
			return true
		default:
			return false
		}
	case StatePublished:
		switch target {
		case StateDeleted:
			return true
		default:
			return false
		}
	default:
		return false
	}
}
