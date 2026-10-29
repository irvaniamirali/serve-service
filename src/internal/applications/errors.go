package applications

import "errors"

var (
	// ErrValidation indicates invalid application input.
	ErrValidation = errors.New("application validation failed")

	// ErrNotFound indicates that the requested application does not exist.
	ErrNotFound = errors.New("application not found")

	// ErrConflict indicates that an application with the same name exists.
	ErrConflict = errors.New("application name already exists")
)
