package httpapi

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/ansel1/merry"
	"github.com/kduong/trading-backend/cmd/storage-service/internal/filestore"
	"github.com/kduong/trading-backend/internal/contextx"
	"github.com/kduong/trading-backend/internal/httpx"
	uuid "github.com/satori/go.uuid"
)

type InitialiseUploadInput struct {
	Filename    string `json:"filename"`
	ContentType string `json:"content_type"`
}

func (input *InitialiseUploadInput) Validate() error {
	if input.Filename == "" {
		return merry.New("filename is required").WithHTTPCode(http.StatusBadRequest)
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
	var input InitialiseUploadInput
	if err = json.NewDecoder(request.Body).Decode(&input); err != nil {
		err = merry.Wrap(err).WithHTTPCode(http.StatusBadRequest)
		return
	}
	if err = input.Validate(); err != nil {
		return
	}
	now := time.Now().UTC().Format(time.RFC3339)
	upload := &filestore.Upload{
		ID:          uuid.NewV4().String(),
		UserID:      contextx.GetUserID(ctx),
		Filename:    input.Filename,
		ContentType: input.ContentType,
		Status:      filestore.UploadStatusInitiated,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err = handler.commandHandler.InitialiseUpload(ctx, upload); err != nil {
		err = merrifyError[err]
		return
	}
	responseWriter.Header().Set("Content-Type", "application/json")
	responseWriter.WriteHeader(http.StatusCreated)
	json.NewEncoder(responseWriter).Encode(upload)
}
