package httpapi

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/ansel1/merry"
	"github.com/gorilla/mux"
	"github.com/kduong-dev/goutil/httpx"
	"github.com/kduong-dev/storage-service/internal/storage"
	"github.com/kduong-dev/storage-service/internal/upload"
)

const maxPartSizeBytes = 5 * 1024 * 1024

type UploadPartResponse struct {
	PartNumber int    `json:"part_number"`
	Size       int64  `json:"size"`
	Checksum   string `json:"checksum"`
}

func (handler *Handler) UploadPart(responseWriter http.ResponseWriter, request *http.Request) {
	var err error
	defer func() {
		if err != nil {
			httpx.SendErrorResponse(responseWriter, err)
		}
	}()
	ctx := request.Context()
	vars := mux.Vars(request)
	uploadID := vars["upload_id"]
	partNumber, err := strconv.Atoi(vars["part_number"])
	if err != nil || partNumber < 1 {
		err = merry.New("part_number must be a positive integer").WithHTTPCode(http.StatusBadRequest)
		return
	}
	if _, err = handler.getActiveUpload(ctx, uploadID); err != nil {
		return
	}
	limitedBody := http.MaxBytesReader(responseWriter, request.Body, maxPartSizeBytes)
	output, err := handler.storage.UploadPart(ctx, storage.UploadPartInput{
		UploadID:   uploadID,
		PartNumber: partNumber,
		Reader:     limitedBody,
	})
	if err != nil {
		var maxBytesError *http.MaxBytesError
		if errors.As(err, &maxBytesError) {
			err = merry.New("part exceeds the 5 MB size limit").WithHTTPCode(http.StatusRequestEntityTooLarge)
		} else {
			err = merry.Wrap(err).WithHTTPCode(http.StatusInternalServerError)
		}
		return
	}
	part := upload.Part{Number: partNumber, Size: output.Size, Checksum: output.Checksum}
	now := time.Now().UTC().Format(time.RFC3339)
	if err = handler.fileInfoStore.RecordPart(ctx, uploadID, part, now); err != nil {
		err = merry.Wrap(err)
		return
	}
	httpx.SendJSONResponse(responseWriter, http.StatusOK, UploadPartResponse{
		PartNumber: partNumber,
		Size:       output.Size,
		Checksum:   output.Checksum,
	})
}
