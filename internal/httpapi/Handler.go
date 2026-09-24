package httpapi

import (
	"context"
	"errors"
	"net/http"

	"github.com/ansel1/merry"
	"github.com/gorilla/mux"
	"github.com/kduong-dev/storage-service/internal/apikey"
	"github.com/kduong-dev/storage-service/internal/fileinfo"
	"github.com/kduong-dev/storage-service/internal/fileinfostore"
	"github.com/kduong-dev/storage-service/internal/storage"
	"github.com/kduong-dev/storage-service/internal/upload"
)

type Handler struct {
	fileInfoStore fileinfostore.Store
	storage       storage.Storage
}

type NewRouterInput struct {
	APIKeyMiddleware *apikey.Middleware
	FileInfoStore    fileinfostore.Store
	Storage          storage.Storage
}

func NewRouter(input NewRouterInput) *mux.Router {
	handler := &Handler{
		fileInfoStore: input.FileInfoStore,
		storage:       input.Storage,
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
	object, err := handler.fileInfoStore.GetUpload(ctx, uploadID)
	if err != nil {
		return nil, merrifyError(err)
	}
	if object.Namespace != apikey.GetNamespace(ctx) {
		return nil, merrifyError(fileinfostore.ErrUploadNotFound)
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
		return nil, merrifyError(fileinfostore.ErrUploadNotActive)
	}
	return object, nil
}

// getFileInfo returns the file info only when it belongs to the caller's namespace.
func (handler *Handler) getFileInfo(ctx context.Context, fileID string) (*fileinfo.FileInfo, error) {
	fileInfo, err := handler.fileInfoStore.GetFileInfo(ctx, fileID)
	if err != nil {
		return nil, merrifyError(err)
	}
	if fileInfo.Namespace != apikey.GetNamespace(ctx) {
		return nil, merrifyError(fileinfostore.ErrFileNotFound)
	}
	return fileInfo, nil
}

func merrifyError(err error) error {
	switch {
	case errors.Is(err, fileinfostore.ErrUploadNotFound):
		return merry.Wrap(err).WithHTTPCode(http.StatusNotFound).WithUserMessage("upload not found")
	case errors.Is(err, fileinfostore.ErrFileNotFound):
		return merry.Wrap(err).WithHTTPCode(http.StatusNotFound).WithUserMessage("file not found")
	case errors.Is(err, fileinfostore.ErrUploadNotActive):
		return merry.Wrap(err).WithHTTPCode(http.StatusConflict).WithUserMessage("upload is not active")
	}
	return err
}
