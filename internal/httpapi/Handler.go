package httpapi

import (
	"context"
	"net/http"
	"strings"

	"github.com/gorilla/mux"
	"github.com/kduong-dev/goutil/httpx"
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
	publicRouter.HandleFunc("/uploads/{upload_id}", handler.GetUploadObject).Methods(http.MethodGet).Name("GetUploadObject")
	publicRouter.HandleFunc("/uploads/{upload_id}/parts/{part_number}", handler.UploadPart).Methods(http.MethodPut).Name("UploadPart")
	publicRouter.HandleFunc("/uploads/{upload_id}/complete", handler.CompleteUpload).Methods(http.MethodPost).Name("CompleteUpload")
	publicRouter.HandleFunc("/uploads/{upload_id}/abort", handler.AbortUpload).Methods(http.MethodPost).Name("AbortUpload")
	publicRouter.HandleFunc("/files", handler.ListFileObjects).Methods(http.MethodGet).Name("ListFileObjects")
	publicRouter.HandleFunc("/files/{file_id}", handler.DownloadFile).Methods(http.MethodGet).Name("DownloadFile")
	publicRouter.HandleFunc("/files/{file_id}/metadata", handler.GetFileObject).Methods(http.MethodGet).Name("GetFileObject")
	publicRouter.HandleFunc("/files/{file_id}/move", handler.MoveFile).Methods(http.MethodPost).Name("MoveFile")
	publicRouter.HandleFunc("/files/{file_id}", handler.DeleteFile).Methods(http.MethodDelete).Name("DeleteFile")
	return router
}

// getUpload returns the upload only when it belongs to the caller's
// namespace; uploads in other namespaces are reported as not found so their
// existence isn't disclosed.
func (handler *Handler) getUpload(ctx context.Context, uploadID string) (*storageservice.UploadObject, error) {
	object, err := handler.uploadObjectStore.Get(ctx, uploadID)
	if err = merrifiedSentinels.MerrifyOrFatal(err); err != nil {
		return nil, err
	}
	if !handler.isInNamespace(ctx, object.Key) {
		return nil, merrifiedSentinels.Merrify(upload.ErrNotFound)
	}
	return object, nil
}

// getFile returns the file only when it belongs to the caller's namespace, for
// the same reason as getUpload.
func (handler *Handler) getFile(ctx context.Context, fileID string) (*storageservice.FileObject, error) {
	object, err := handler.fileObjectStore.Get(ctx, fileID)
	if err = merrifiedSentinels.MerrifyOrFatal(err); err != nil {
		return nil, err
	}
	if !handler.isInNamespace(ctx, object.Key) {
		return nil, merrifiedSentinels.Merrify(file.ErrNotFound)
	}
	return object, nil
}

// isInNamespace reports whether the key sits under the caller's namespace.
// The trailing slash stops namespace "alpha" from matching "alpha-service/".
func (handler *Handler) isInNamespace(ctx context.Context, key string) bool {
	return strings.HasPrefix(key, apikey.GetNamespace(ctx)+"/")
}

var merrifiedSentinels = httpx.MerrifiedSentinels{
	{Sentinel: upload.ErrNotFound, StatusCode: http.StatusNotFound, UserMessage: "upload not found"},
	{Sentinel: storage.ErrUploadNotFound, StatusCode: http.StatusNotFound, UserMessage: "upload not found"},
	{Sentinel: file.ErrNotFound, StatusCode: http.StatusNotFound, UserMessage: "file not found"},
	{Sentinel: file.ErrInvalidAfter, StatusCode: http.StatusBadRequest, UserMessage: "invalid cursor"},
}
