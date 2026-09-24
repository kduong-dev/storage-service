package httpapi

import (
	"net/http"
	"sort"
	"time"

	"github.com/ansel1/merry"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/kduong-dev/goutil/fatal"
	"github.com/kduong-dev/goutil/httpx"
	"github.com/kduong-dev/storage-service/internal/storage"
	"github.com/kduong-dev/storage-service/internal/upload"
	"github.com/kduong-dev/storage-service/pkg/storageservice"
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
	uploadObject, err := handler.getUpload(ctx, uploadID)
	if err != nil {
		return
	}
	if len(uploadObject.Parts) == 0 {
		err = merry.UserError("no parts have been uploaded").WithHTTPCode(http.StatusBadRequest)
		return
	}
	partNumbers := make([]int, len(uploadObject.Parts))
	for index, part := range uploadObject.Parts {
		partNumbers[index] = part.PartNumber
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
		err = merrify(err)
		return
	}
	now := time.Now().UTC().Format(time.RFC3339)
	err = handler.uploadObjectStore.Complete(ctx, upload.CompleteInput{
		UploadID:  uploadID,
		FileID:    fileID,
		Size:      output.Size,
		Checksum:  output.Checksum,
		UpdatedAt: now,
	})
	if err = merrifyOrFatal(err); err != nil {
		return
	}
	fileObject := &storageservice.File{
		ID:          fileID,
		UploadID:    uploadID,
		Key:         uploadObject.Key,
		ContentType: uploadObject.ContentType,
		Size:        output.Size,
		Checksum:    output.Checksum,
		CreatedAt:   now,
	}
	fatal.OnError(handler.fileObjectStore.Put(ctx, fileObject))
	httpx.SendJSONResponse(responseWriter, http.StatusCreated, fileObject)
}
