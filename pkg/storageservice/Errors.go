package storageservice

import "errors"

var (
	ErrUnauthorized        = errors.New("unauthorized")
	ErrUploadNotFound      = errors.New("upload not found")
	ErrFileNotFound        = errors.New("file not found")
	ErrBadRequest          = errors.New("bad request")
	ErrRangeNotSatisfiable = errors.New("range not satisfiable")
	ErrServerError         = errors.New("server error")
)
