package httpapi

import (
	"context"
	"errors"
	"net/http"

	"github.com/ansel1/merry"
	"github.com/gorilla/mux"
	"github.com/kduong-dev/storage-service/internal/apikey"
	"github.com/kduong-dev/storage-service/internal/filestore"
	"github.com/kduong-dev/storage-service/internal/storage"
)

type Handler struct {
	commandHandler filestore.CommandHandler
	queryHandler   filestore.QueryHandler
	backend        storage.Backend
}

type NewRouterInput struct {
	APIKeyMiddleware *apikey.Middleware
	CommandHandler   filestore.CommandHandler
	QueryHandler     filestore.QueryHandler
	Backend          storage.Backend
}

func NewRouter(input NewRouterInput) *mux.Router {
	handler := &Handler{
		commandHandler: input.CommandHandler,
		queryHandler:   input.QueryHandler,
		backend:        input.Backend,
	}
	router := mux.NewRouter().StrictSlash(true)
	v1 := router.PathPrefix("/storage/v1").Subrouter()
	v1.Use(input.APIKeyMiddleware.Handle)
	v1.HandleFunc("/uploads", handler.InitialiseUpload).Methods(http.MethodPost).Name("InitialiseUpload")
	v1.HandleFunc("/uploads/{upload_id}/parts/{part_number}", handler.UploadPart).Methods(http.MethodPut).Name("UploadPart")
	v1.HandleFunc("/uploads/{upload_id}/complete", handler.CompleteUpload).Methods(http.MethodPost).Name("CompleteUpload")
	v1.HandleFunc("/files/{file_id}", handler.DownloadFile).Methods(http.MethodGet).Name("DownloadFile")
	return router
}

// getUpload returns the upload only when it belongs to the caller's
// namespace; uploads in other namespaces are reported as not found so their
// existence isn't disclosed.
func (handler *Handler) getUpload(ctx context.Context, uploadID string) (*filestore.Upload, error) {
	upload, err := handler.queryHandler.GetUpload(ctx, uploadID)
	if err != nil {
		return nil, merrifyError(err)
	}
	if upload.Namespace != apikey.GetNamespace(ctx) {
		return nil, merrifyError(filestore.ErrUploadNotFound)
	}
	return upload, nil
}

// getFile returns the file only when it belongs to the caller's namespace.
func (handler *Handler) getFile(ctx context.Context, fileID string) (*filestore.File, error) {
	file, err := handler.queryHandler.GetFile(ctx, fileID)
	if err != nil {
		return nil, merrifyError(err)
	}
	if file.Namespace != apikey.GetNamespace(ctx) {
		return nil, merrifyError(filestore.ErrFileNotFound)
	}
	return file, nil
}

func merrifyError(err error) error {
	switch {
	case errors.Is(err, filestore.ErrUploadNotFound):
		return merry.Wrap(err).WithHTTPCode(http.StatusNotFound).WithUserMessage("upload not found")
	case errors.Is(err, filestore.ErrFileNotFound):
		return merry.Wrap(err).WithHTTPCode(http.StatusNotFound).WithUserMessage("file not found")
	case errors.Is(err, filestore.ErrUploadNotActive):
		return merry.Wrap(err).WithHTTPCode(http.StatusConflict).WithUserMessage("upload is not active")
	}
	return err
}
