package httpapi

import (
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

var merrifyError = map[error]error{
	filestore.ErrUploadNotFound:  merry.New("upload not found").WithHTTPCode(http.StatusNotFound),
	filestore.ErrUploadForbidden: merry.New("forbidden").WithHTTPCode(http.StatusForbidden),
	filestore.ErrFileNotFound:    merry.New("file not found").WithHTTPCode(http.StatusNotFound),
	filestore.ErrFileForbidden:   merry.New("forbidden").WithHTTPCode(http.StatusForbidden),
	filestore.ErrUploadNotActive: merry.New("upload is not active").WithHTTPCode(http.StatusConflict),
}
