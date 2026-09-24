package httpapi

import (
	"context"
	"net/http"
	"strings"

	"github.com/gorilla/mux"
	"github.com/kduong-dev/storage-service/internal/apikey"
	"github.com/kduong-dev/storage-service/internal/file"
	"github.com/kduong-dev/storage-service/internal/storage"
	"github.com/kduong-dev/storage-service/internal/upload"
	"github.com/kduong-dev/storage-service/pkg/storageservice"
)

type Handler struct {
	uploadObjectStore upload.ObjectStore
	fileObjectStore   file.ObjectStore
	storage           storage.Storage
}

type NewRouterInput struct {
	APIKeyMiddleware  *apikey.Middleware
	UploadObjectStore upload.ObjectStore
	FileObjectStore   file.ObjectStore
	Storage           storage.Storage
}

func NewRouter(input NewRouterInput) *mux.Router {
	handler := &Handler{
		uploadObjectStore: input.UploadObjectStore,
		fileObjectStore:   input.FileObjectStore,
		storage:           input.Storage,
	}
	router := mux.NewRouter().StrictSlash(true)
	publicRouter := router.PathPrefix("/storage/v1").Subrouter()
	publicRouter.Use(input.APIKeyMiddleware.Handle)
	publicRouter.HandleFunc("/uploads", handler.InitialiseUpload).Methods(http.MethodPost).Name("InitialiseUpload")
	publicRouter.HandleFunc("/uploads/{upload_id}/parts/{part_number}", handler.UploadPart).Methods(http.MethodPut).Name("UploadPart")
	publicRouter.HandleFunc("/uploads/{upload_id}/complete", handler.CompleteUpload).Methods(http.MethodPost).Name("CompleteUpload")
	publicRouter.HandleFunc("/uploads/{upload_id}/abort", handler.AbortUpload).Methods(http.MethodPost).Name("AbortUpload")
	publicRouter.HandleFunc("/files", handler.ListFiles).Methods(http.MethodGet).Name("ListFiles")
	publicRouter.HandleFunc("/files/{file_id}", handler.DownloadFile).Methods(http.MethodGet).Name("DownloadFile")
	return router
}

// getUpload returns the upload only when it belongs to the caller's
// namespace; uploads in other namespaces are reported as not found so their
// existence isn't disclosed.
func (handler *Handler) getUpload(ctx context.Context, uploadID string) (*storageservice.Upload, error) {
	object, err := handler.uploadObjectStore.Get(ctx, uploadID)
	if err = toResponseErrorOrFatal(err); err != nil {
		return nil, err
	}
	if !handler.isInNamespace(ctx, object.Key) {
		return nil, toResponseError(upload.ErrNotFound)
	}
	return object, nil
}

// isInNamespace reports whether the key sits under the caller's namespace.
// The trailing slash stops namespace "alpha" from matching "alpha-service/".
func (handler *Handler) isInNamespace(ctx context.Context, key string) bool {
	return strings.HasPrefix(key, apikey.GetNamespace(ctx)+"/")
}
