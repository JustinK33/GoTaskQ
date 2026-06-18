package store

import "errors"

var (
	ErrJobNotFound       = errors.New("store: job not found")
	ErrInvalidTransition = errors.New("store: invalid state transition")
)
