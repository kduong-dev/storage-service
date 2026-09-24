package httpapi

import (
	"errors"
	"net/http"

	"github.com/ansel1/merry"
	"github.com/kduong-dev/goutil/fatal"
	"github.com/kduong-dev/storage-service/internal/file"
	"github.com/kduong-dev/storage-service/internal/storage"
	"github.com/kduong-dev/storage-service/internal/upload"
)

// responseErrors are the store and storage sentinels a client can cause,
// with the status code and user message each is sent as.
var responseErrors = []struct {
	sentinel    error
	statusCode  int
	userMessage string
}{
	{upload.ErrNotFound, http.StatusNotFound, "upload not found"},
	{storage.ErrUploadNotFound, http.StatusNotFound, "upload not found"},
	{file.ErrNotFound, http.StatusNotFound, "file not found"},
	{file.ErrInvalidAfter, http.StatusBadRequest, "invalid cursor"},
}

func lookupResponseError(err error) error {
	for _, responseError := range responseErrors {
		if errors.Is(err, responseError.sentinel) {
			return merry.Wrap(err).WithHTTPCode(responseError.statusCode).WithUserMessage(responseError.userMessage)
		}
	}
	return nil
}

// toResponseError merrifies err when it matches one of responseErrors, and
// otherwise returns it unchanged.
func toResponseError(err error) error {
	if responseError := lookupResponseError(err); responseError != nil {
		return responseError
	}
	return err
}

// toResponseErrorOrFatal is toResponseError for calls whose only expected
// failures are responseErrors.
func toResponseErrorOrFatal(err error) error {
	responseError := lookupResponseError(err)
	if responseError == nil {
		fatal.OnError(err)
	}
	return responseError
}
