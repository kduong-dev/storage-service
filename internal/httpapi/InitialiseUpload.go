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

// validateKey rejects keys that could climb out of the caller's namespace,
// since keys become paths under it.
func validateKey(key string) error {
	if !filepath.IsLocal(key) || strings.Contains(key, `\`) {
		return merry.UserError("key must be a relative path without '..' segments").WithHTTPCode(http.StatusBadRequest)
	}
	return nil
}

func validateInitialiseUploadRequest(input storageservice.InitialiseUploadInput) error {
	if err := validateKey(input.Key); err != nil {
		return err
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
	object := &storageservice.UploadObject{
		ID:          uploadID,
		Key:         apikey.GetNamespace(ctx) + "/" + input.Key,
		ContentType: input.ContentType,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	fatal.OnError(handler.uploadObjectStore.Initialise(ctx, object))
	httpx.SendJSONResponse(responseWriter, http.StatusCreated, object)
}
