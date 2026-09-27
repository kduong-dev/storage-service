package httpapi

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/kduong-dev/storage-service/internal/apikey"
	"github.com/kduong-dev/storage-service/internal/file"
	"github.com/kduong-dev/storage-service/internal/storage"
	"github.com/kduong-dev/storage-service/internal/upload"
)

type NewHandlerInput struct {
	APIKeyMiddleware  *apikey.Middleware
	UploadObjectStore upload.ObjectStore
	FileObjectStore   file.ObjectStore
	Storage           storage.Storage
}

func NewHandler(input NewHandlerInput) http.Handler {
	api := &API{
		uploadObjectStore: input.UploadObjectStore,
		fileObjectStore:   input.FileObjectStore,
		storage:           input.Storage,
	}
	router := mux.NewRouter().StrictSlash(true)
	publicRouter := router.PathPrefix("/storage/v1").Subrouter()
	publicRouter.Use(input.APIKeyMiddleware.Handle)
	publicRouter.HandleFunc("/uploads", api.InitialiseUpload).Methods(http.MethodPost).Name("InitialiseUpload")
	publicRouter.HandleFunc("/uploads/{upload_id}", api.GetUploadObject).Methods(http.MethodGet).Name("GetUploadObject")
	publicRouter.HandleFunc("/uploads/{upload_id}/parts/{part_number}", api.UploadPart).Methods(http.MethodPut).Name("UploadPart")
	publicRouter.HandleFunc("/uploads/{upload_id}/complete", api.CompleteUpload).Methods(http.MethodPost).Name("CompleteUpload")
	publicRouter.HandleFunc("/uploads/{upload_id}/abort", api.AbortUpload).Methods(http.MethodPost).Name("AbortUpload")
	publicRouter.HandleFunc("/files", api.ListFileObjects).Methods(http.MethodGet).Name("ListFileObjects")
	publicRouter.HandleFunc("/files/{file_id}", api.DownloadFile).Methods(http.MethodGet).Name("DownloadFile")
	publicRouter.HandleFunc("/files/{file_id}/metadata", api.GetFileObject).Methods(http.MethodGet).Name("GetFileObject")
	publicRouter.HandleFunc("/files/{file_id}/move", api.MoveFile).Methods(http.MethodPost).Name("MoveFile")
	publicRouter.HandleFunc("/files/{file_id}", api.DeleteFile).Methods(http.MethodDelete).Name("DeleteFile")
	return router
}
