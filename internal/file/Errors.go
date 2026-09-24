package file

import "errors"

var (
	ErrUploadNotFound  = errors.New("upload not found")
	ErrFileNotFound    = errors.New("file not found")
	ErrUploadNotActive = errors.New("upload is not in an active state")
)
