package httpapi

import (
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/ansel1/merry"
	"github.com/google/uuid"
	"github.com/kduong-dev/goutil/httpx"
	"github.com/kduong-dev/storage-service/internal/apikey"
	"github.com/kduong-dev/storage-service/internal/upload"
)

type InitialiseUploadInput struct {
	Key         string `json:"key"`
	ContentType string `json:"content_type"`
}

func (input *InitialiseUploadInput) Validate() error {
	// Keys become paths under the caller's namespace, so they must not be able
	// to climb out of it.
	if !filepath.IsLocal(input.Key) || strings.Contains(input.Key, `\`) {
		return merry.New("key must be a relative path without '..' segments").WithHTTPCode(http.StatusBadRequest)
	}
	if input.ContentType == "" {
		return merry.New("content_type is required").WithHTTPCode(http.StatusBadRequest)
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
	input, err := httpx.DecodeJSONBody[InitialiseUploadInput](request)
	if err != nil {
		return
	}
	if err = input.Validate(); err != nil {
		return
	}
	now := time.Now().UTC().Format(time.RFC3339)
	object := &upload.Object{
		ID:          uuid.NewString(),
		Key:         apikey.GetNamespace(ctx) + "/" + input.Key,
		ContentType: input.ContentType,
		Status:      upload.StatusInitiated,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err = handler.uploadObjectStore.Initialise(ctx, object); err != nil {
		err = merrifyError(err)
		return
	}
	if err = handler.storage.InitialiseUpload(ctx, object.ID); err != nil {
		return
	}
	httpx.SendJSONResponse(responseWriter, http.StatusCreated, object)
}
