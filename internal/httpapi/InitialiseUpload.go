package httpapi

import (
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/ansel1/merry"
	"github.com/google/uuid"
	"github.com/kduong-dev/goutil/fatal"
	"github.com/kduong-dev/goutil/httpx"
	"github.com/kduong-dev/storage-service/internal/apikey"
	"github.com/kduong-dev/storage-service/pkg/storageservice"
)

func validateInitialiseUploadRequest(input storageservice.InitialiseUploadInput) error {
	// Keys become paths under the caller's namespace, so they must not be able
	// to climb out of it.
	if !filepath.IsLocal(input.Key) || strings.Contains(input.Key, `\`) {
		return merry.UserError("key must be a relative path without '..' segments").WithHTTPCode(http.StatusBadRequest)
	}
	if input.ContentType == "" {
		return merry.UserError("content_type is required").WithHTTPCode(http.StatusBadRequest)
	}
	return nil
}

func (handler *Handler) InitialiseUpload(responseWriter http.ResponseWriter, request *http.Request) {
	var err error
	defer func() {
		if err != nil {
			httpx.SendErrorResponse(responseWriter, err)
		}
	}()
	ctx := request.Context()
	input, err := httpx.DecodeJSONBody[storageservice.InitialiseUploadInput](request)
	if err != nil {
		return
	}
	if err = validateInitialiseUploadRequest(input); err != nil {
		return
	}
	uploadID := uuid.NewString()
	err = handler.storage.InitialiseUpload(ctx, uploadID)
	if err != nil {
		return
	}
	now := time.Now().UTC().Format(time.RFC3339)
	object := &storageservice.Upload{
		ID:          uploadID,
		Key:         apikey.GetNamespace(ctx) + "/" + input.Key,
		ContentType: input.ContentType,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	fatal.OnError(handler.uploadObjectStore.Initialise(ctx, object))
	httpx.SendJSONResponse(responseWriter, http.StatusCreated, object)
}
