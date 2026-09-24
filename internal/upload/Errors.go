package upload

import "errors"

var (
	ErrNotFound  = errors.New("upload not found")
	ErrNotActive = errors.New("upload is not in an active state")
)
