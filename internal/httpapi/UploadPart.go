package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/ansel1/merry"
	"github.com/gorilla/mux"
	"github.com/kduong/trading-backend/cmd/storage-service/internal/filestore"
	"github.com/kduong/trading-backend/internal/httputil"
)

type UploadPartResponse struct {
	PartNumber int    `json:"part_number"`
	Size       int64  `json:"size"`
	Checksum   string `json:"checksum"`
}

func (handler *Handler) UploadPart(responseWriter http.ResponseWriter, request *http.Request) {
	var err error
	defer func() {
		if err != nil {
			httputil.SendErrorResponse(responseWriter, err)
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
	// Verify ownership via the query handler before accepting bytes.
	if _, err = handler.queryHandler.GetUpload(ctx, uploadID); err != nil {
		err = merrifyError[err]
		return
	}
	const maxPartSize = 5 * 1024 * 1024 // 5 MB
	limitedBody := http.MaxBytesReader(responseWriter, request.Body, maxPartSize)
	size, checksum, err := handler.backend.WritePart(uploadID, partNumber, limitedBody)
	if err != nil {
		var maxBytesError *http.MaxBytesError
		if errors.As(err, &maxBytesError) {
			err = merry.New("part exceeds the 5 MB size limit").WithHTTPCode(http.StatusRequestEntityTooLarge)
		} else {
			err = merry.Wrap(err).WithHTTPCode(http.StatusInternalServerError)
		}
		return
	}

	part := filestore.Part{PartNumber: partNumber, Size: size, Checksum: checksum}
	now := time.Now().UTC().Format(time.RFC3339)
	if err = handler.commandHandler.RecordPart(ctx, uploadID, part, now); err != nil {
		err = merrifyError[err]
		return
	}

	responseWriter.Header().Set("Content-Type", "application/json")
	responseWriter.WriteHeader(http.StatusOK)
	json.NewEncoder(responseWriter).Encode(UploadPartResponse{
		PartNumber: partNumber,
		Size:       size,
		Checksum:   checksum,
	})
}
