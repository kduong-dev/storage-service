package file

import "errors"

var (
	ErrNotFound      = errors.New("file not found")
	ErrAlreadyExists = errors.New("file already exists")
)
