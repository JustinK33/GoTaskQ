package store

import "errors"

var (
	// ErrJobNotFound is returned by GetJob when no row matches the requested id.
	ErrJobNotFound = errors.New("store: job not found")

	// ErrInvalidTransition is returned by UpdateJob when CanTransition rejects the new state.
	ErrInvalidTransition = errors.New("store: invalid state transition")
)
