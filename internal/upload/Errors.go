package upload

import "errors"

var (
	ErrNotFound      = errors.New("upload not found")
	ErrAlreadyExists = errors.New("upload already exists")
	ErrNotActive     = errors.New("upload is not in an active state")
)
