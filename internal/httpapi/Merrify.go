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

type merrifiedSentinel struct {
	sentinel    error
	statusCode  int
	userMessage string
}

// merrifiedSentinels are the store and storage sentinels a client can cause,
// with the status code and user message each is sent as.
var merrifiedSentinels = []merrifiedSentinel{
	{upload.ErrNotFound, http.StatusNotFound, "upload not found"},
	{storage.ErrUploadNotFound, http.StatusNotFound, "upload not found"},
	{file.ErrNotFound, http.StatusNotFound, "file not found"},
	{file.ErrInvalidAfter, http.StatusBadRequest, "invalid cursor"},
}

func lookupMerrifiedSentinel(err error) error {
	for _, merrified := range merrifiedSentinels {
		if errors.Is(err, merrified.sentinel) {
			return merry.Wrap(err).WithHTTPCode(merrified.statusCode).WithUserMessage(merrified.userMessage)
		}
	}
	return nil
}

// merrify attaches the status code and user message of the matching
// merrifiedSentinels entry to err, and otherwise returns it unchanged.
func merrify(err error) error {
	if merrifiedError := lookupMerrifiedSentinel(err); merrifiedError != nil {
		return merrifiedError
	}
	return err
}

// merrifyOrFatal is merrify for calls whose only expected failures are
// merrifiedSentinels.
func merrifyOrFatal(err error) error {
	merrifiedError := lookupMerrifiedSentinel(err)
	if merrifiedError == nil {
		fatal.OnError(err)
	}
	return merrifiedError
}
