package httpapi

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/ansel1/merry"
	"github.com/gorilla/mux"
	"github.com/kduong-dev/goutil/fatal"
	"github.com/kduong-dev/storage-service/internal/apikey"
	"github.com/kduong-dev/storage-service/internal/file"
	"github.com/kduong-dev/storage-service/internal/storage"
	"github.com/kduong-dev/storage-service/internal/upload"
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
	publicRouter.HandleFunc("/files/{file_id}", handler.DownloadFile).Methods(http.MethodGet).Name("DownloadFile")
	return router
}

// getUpload returns the upload only when it belongs to the caller's
// namespace; uploads in other namespaces are reported as not found so their
// existence isn't disclosed.
func (handler *Handler) getUpload(ctx context.Context, uploadID string) (*upload.Object, error) {
	object, err := handler.uploadObjectStore.Get(ctx, uploadID)
	if errors.Is(err, upload.ErrNotFound) {
		return nil, merry.Wrap(err).WithHTTPCode(http.StatusNotFound).WithUserMessage("upload not found")
	}
	fatal.OnError(err)
	if !isInNamespace(ctx, object.Key) {
		return nil, merry.Wrap(upload.ErrNotFound).WithHTTPCode(http.StatusNotFound).WithUserMessage("upload not found")
	}
	return object, nil
}

// getActiveUpload is getUpload for uploads still accepting changes, checked
// before touching storage since inactive uploads no longer have parts on disk.
func (handler *Handler) getActiveUpload(ctx context.Context, uploadID string) (*upload.Object, error) {
	object, err := handler.getUpload(ctx, uploadID)
	if err != nil {
		return nil, err
	}
	if object.Status != upload.StatusInitiated {
		return nil, merry.Wrap(upload.ErrNotActive).WithHTTPCode(http.StatusConflict).WithUserMessage("upload is not active")
	}
	return object, nil
}

func (handler *Handler) getFile(ctx context.Context, fileID string) (*file.Object, error) {
	object, err := handler.fileObjectStore.Get(ctx, fileID)
	if errors.Is(err, file.ErrNotFound) {
		return nil, merry.Wrap(err).WithHTTPCode(http.StatusNotFound).WithUserMessage("file not found")
	}
	fatal.OnError(err)
	if !isInNamespace(ctx, object.Key) {
		return nil, merry.Wrap(file.ErrNotFound).WithHTTPCode(http.StatusNotFound).WithUserMessage("file not found")
	}
	return object, nil
}

// isInNamespace reports whether the key sits under the caller's namespace.
// The trailing slash stops namespace "alpha" from matching "alpha-service/".
func isInNamespace(ctx context.Context, key string) bool {
	return strings.HasPrefix(key, apikey.GetNamespace(ctx)+"/")
}
