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
	uploadObject, err := handler.getActiveUpload(ctx, uploadID)
	if err != nil {
		return
	}
	if len(uploadObject.Parts) == 0 {
		err = merry.New("no parts have been uploaded").WithHTTPCode(http.StatusBadRequest)
		return
	}
	partNumbers := make([]int, len(uploadObject.Parts))
	for index, part := range uploadObject.Parts {
		partNumbers[index] = part.Number
	}
	sort.Ints(partNumbers)
	fileID := uuid.NewString()
	output, err := handler.storage.CompleteUpload(ctx, storage.CompleteUploadInput{
		UploadID:    uploadID,
		FileID:      fileID,
		Key:         uploadObject.Key,
		PartNumbers: partNumbers,
	})
	if err != nil {
		return
	}
	now := time.Now().UTC().Format(time.RFC3339)
	if err = handler.uploadObjectStore.Complete(ctx, uploadID, now); err != nil {
		err = merrifyError(err)
		return
	}
	fileObject := &file.Object{
		ID:          fileID,
		UploadID:    uploadID,
		Key:         uploadObject.Key,
		ContentType: uploadObject.ContentType,
		Size:        output.Size,
		Checksum:    output.Checksum,
		CreatedAt:   now,
	}
	if err = handler.fileObjectStore.Put(ctx, fileObject); err != nil {
		err = merry.Wrap(err)
		return
	}
	httpx.SendJSONResponse(responseWriter, http.StatusCreated, fileObject)
}
