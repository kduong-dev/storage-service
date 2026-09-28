package httpapi

import (
	"context"
	"net/http"
	"strings"

	"github.com/kduong-dev/goutil/httpx"
	"github.com/kduong-dev/storage-service/internal/apikey"
	"github.com/kduong-dev/storage-service/internal/file"
	"github.com/kduong-dev/storage-service/internal/storage"
	"github.com/kduong-dev/storage-service/internal/upload"
	"github.com/kduong-dev/storage-service/pkg/storageservice"
)

type API struct {
	uploadObjectStore upload.ObjectStore
	fileObjectStore   file.ObjectStore
	storage           storage.Storage
}

// getUpload returns the upload only when it belongs to the caller's
// namespace; uploads in other namespaces are reported as not found so their
// existence isn't disclosed.
func (api *API) getUpload(ctx context.Context, uploadID string) (*storageservice.UploadObject, error) {
	object, err := api.uploadObjectStore.Get(ctx, uploadID)
	if err != nil {
		return nil, err
	}
	if !api.isInNamespace(ctx, object.Key) {
		return nil, upload.ErrNotFound
	}
	return object, nil
}

// getFile returns the file only when it belongs to the caller's namespace, for
// the same reason as getUpload.
func (api *API) getFile(ctx context.Context, fileID string) (*storageservice.FileObject, error) {
	object, err := api.fileObjectStore.Get(ctx, fileID)
	if err != nil {
		return nil, err
	}
	if !api.isInNamespace(ctx, object.Key) {
		return nil, file.ErrNotFound
	}
	return object, nil
}

// isInNamespace reports whether the key sits under the caller's namespace.
// The trailing slash stops namespace "alpha" from matching "alpha-service/".
func (api *API) isInNamespace(ctx context.Context, key string) bool {
	return strings.HasPrefix(key, apikey.GetNamespace(ctx)+"/")
}

var merrifiedSentinels = httpx.MerrifiedSentinels{
	{Sentinel: upload.ErrNotFound, StatusCode: http.StatusNotFound, UserMessage: "upload not found"},
	{Sentinel: storage.ErrUploadNotFound, StatusCode: http.StatusNotFound, UserMessage: "upload not found"},
	{Sentinel: file.ErrNotFound, StatusCode: http.StatusNotFound, UserMessage: "file not found"},
	{Sentinel: file.ErrInvalidAfter, StatusCode: http.StatusBadRequest, UserMessage: "invalid cursor"},
}
