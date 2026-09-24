package storageservice

import "errors"

var (
	ErrUnauthorized   = errors.New("unauthorized")
	ErrUploadNotFound = errors.New("upload not found")
	ErrFileNotFound   = errors.New("file not found")
	ErrBadRequest     = errors.New("bad request")
	ErrServerError    = errors.New("server error")
)
