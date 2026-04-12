package filestore

import "errors"

var (
	ErrUploadNotFound  = errors.New("upload not found")
	ErrUploadForbidden = errors.New("upload forbidden")
	ErrFileNotFound    = errors.New("file not found")
	ErrFileForbidden   = errors.New("file forbidden")
	ErrUploadNotActive = errors.New("upload is not in an active state")
)
