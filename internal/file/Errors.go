package file

import "errors"

var (
	ErrNotFound      = errors.New("file not found")
	ErrAlreadyExists = errors.New("file already exists")
	ErrInvalidAfter  = errors.New("after does not refer to an object under the key prefix")
)
