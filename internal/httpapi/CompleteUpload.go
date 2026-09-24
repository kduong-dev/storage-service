package httpapi

import (
	"net/http"
	"sort"
	"time"

	"github.com/ansel1/merry"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/kduong-dev/goutil/httpx"
	"github.com/kduong-dev/storage-service/internal/file"
	"github.com/kduong-dev/storage-service/internal/storage"
)

func (handler *Handler) CompleteUpload(responseWriter http.ResponseWriter, request *http.Request) {
	var err error
	defer func() {
		if err != nil {
			httpx.SendErrorResponse(responseWriter, err)
		}
	}()
	ctx := request.Context()
	vars := mux.Vars(request)
	uploadID := vars["upload_id"]
	upload, err := handler.getActiveUpload(ctx, uploadID)
	if err != nil {
		return
	}
	if len(upload.Parts) == 0 {
		err = merry.New("no parts have been uploaded").WithHTTPCode(http.StatusBadRequest)
		return
	}
	partNumbers := make([]int, len(upload.Parts))
	for index, part := range upload.Parts {
		partNumbers[index] = part.Number
	}
	sort.Ints(partNumbers)
	fileID := uuid.NewString()
	output, err := handler.storage.CompleteUpload(ctx, storage.CompleteUploadInput{
		UploadID:    uploadID,
		FileID:      fileID,
		Key:         upload.Key,
		PartNumbers: partNumbers,
	})
	if err != nil {
		return
	}
	err = handler.fileInfoStore.CompleteUpload(ctx, file.CompleteUploadInput{
		UploadID:  uploadID,
		FileID:    fileID,
		Size:      output.Size,
		Checksum:  output.Checksum,
		UpdatedAt: time.Now().UTC().Format(time.RFC3339),
	})
	if err != nil {
		err = merry.Wrap(err)
		return
	}
	fileInfo, err := handler.getFileInfo(ctx, fileID)
	if err != nil {
		return
	}
	httpx.SendJSONResponse(responseWriter, http.StatusCreated, fileInfo)
}
