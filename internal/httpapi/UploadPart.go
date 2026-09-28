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
	"github.com/kduong-dev/storage-service/pkg/storageservice"
)

func (api *API) UploadPart(responseWriter http.ResponseWriter, request *http.Request) {
	var err error
	defer func() {
		if err != nil {
			merrifiedSentinels.SendErrorResponse(responseWriter, err)
		}
	}()
	ctx := request.Context()
	vars := mux.Vars(request)
	uploadID := vars["upload_id"]
	partNumber, err := strconv.Atoi(vars["part_number"])
	if err != nil || partNumber < 1 {
		err = merry.UserError("part_number must be a positive integer").WithHTTPCode(http.StatusBadRequest)
		return
	}
	if _, err = api.getUpload(ctx, uploadID); err != nil {
		return
	}
	limitedBody := http.MaxBytesReader(responseWriter, request.Body, storageservice.MaxPartSizeBytes)
	output, err := api.storage.UploadPart(ctx, storage.UploadPartInput{
		UploadID:   uploadID,
		PartNumber: partNumber,
		Reader:     limitedBody,
	})
	if err != nil {
		var maxBytesError *http.MaxBytesError
		if errors.As(err, &maxBytesError) {
			err = merry.UserErrorf("part exceeds the %d MB size limit", storageservice.MaxPartSizeBytes/(1024*1024)).WithHTTPCode(http.StatusRequestEntityTooLarge)
		}
		return
	}
	err = api.uploadObjectStore.RecordPart(ctx, upload.RecordPartInput{
		UploadID:  uploadID,
		Part:      storageservice.Part{PartNumber: partNumber, Size: output.Size, Checksum: output.Checksum},
		UpdatedAt: time.Now().UTC().Format(time.RFC3339),
	})
	if err != nil {
		return
	}
	httpx.SendJSONResponse(responseWriter, http.StatusOK, storageservice.UploadPartOutput{
		PartNumber: partNumber,
		Size:       output.Size,
		Checksum:   output.Checksum,
	})
}
