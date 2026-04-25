package httpapi

import (
	"errors"
	"net/http"

	"github.com/ansel1/merry"
	"github.com/gorilla/mux"
	"github.com/kduong/trading-backend/cmd/storage-service/internal/filestore"
	"github.com/kduong/trading-backend/cmd/storage-service/internal/storage"
	"github.com/kduong/trading-backend/internal/auth"
)

type Handler struct {
	commandHandler filestore.CommandHandler
	queryHandler   filestore.QueryHandler
	backend        storage.Backend
}

type NewRouterInput struct {
	AuthMiddleware *auth.Middleware
	CommandHandler filestore.CommandHandler
	QueryHandler   filestore.QueryHandler
	Backend        storage.Backend
}

func NewRouter(input NewRouterInput) *mux.Router {
	handler := &Handler{
		commandHandler: input.CommandHandler,
		queryHandler:   input.QueryHandler,
		backend:        input.Backend,
	}
	router := mux.NewRouter().StrictSlash(true)
	v1 := router.PathPrefix("/storage/v1").Subrouter()
	v1.Use(input.AuthMiddleware.Handle)
	v1.HandleFunc("/uploads", handler.InitialiseUpload).Methods(http.MethodPost).Name("InitialiseUpload")
	v1.HandleFunc("/uploads/{upload_id}/parts/{part_number}", handler.UploadPart).Methods(http.MethodPut).Name("UploadPart")
	v1.HandleFunc("/uploads/{upload_id}/complete", handler.CompleteUpload).Methods(http.MethodPost).Name("CompleteUpload")
	v1.HandleFunc("/files/{file_id}", handler.DownloadFile).Methods(http.MethodGet).Name("DownloadFile")
	return router
}

func merrifyError(err error) error {
	switch {
	case errors.Is(err, filestore.ErrUploadNotFound):
		return merry.Wrap(err).WithHTTPCode(http.StatusNotFound).WithUserMessage("upload not found")
	case errors.Is(err, filestore.ErrUploadForbidden):
		return merry.Wrap(err).WithHTTPCode(http.StatusForbidden).WithUserMessage("forbidden")
	case errors.Is(err, filestore.ErrFileNotFound):
		return merry.Wrap(err).WithHTTPCode(http.StatusNotFound).WithUserMessage("file not found")
	case errors.Is(err, filestore.ErrFileForbidden):
		return merry.Wrap(err).WithHTTPCode(http.StatusForbidden).WithUserMessage("forbidden")
	case errors.Is(err, filestore.ErrUploadNotActive):
		return merry.Wrap(err).WithHTTPCode(http.StatusConflict).WithUserMessage("upload is not active")
	}
	return err
}
