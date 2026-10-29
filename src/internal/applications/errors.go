package applications

import "errors"

var (
	// ErrValidation indicates invalid application input.
	ErrValidation = errors.New("application validation failed")

	// ErrNotFound indicates that the requested application does not exist.
	ErrNotFound = errors.New("application not found")

	// ErrConflict indicates that an application registration already exists.
	ErrConflict = errors.New("application registration already exists")
)
