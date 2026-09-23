package httpapi

import (
	"net/http"
	"sort"
	"time"

	"github.com/ansel1/merry"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/kduong-dev/goutil/httpx"
	"github.com/kduong-dev/goutil/logx"
	"github.com/kduong-dev/storage-service/internal/filestore"
)

func (handler *Handler) CompleteUpload(responseWriter http.ResponseWriter, request *http.Request) {
	var err error
	defer func() {
		if err != nil {
			httpx.SendErrorResponse(responseWriter, err)
		}
	}()
	ctx := request.Context()
	uploadID := mux.Vars(request)["upload_id"]
	upload, err := handler.getUpload(ctx, uploadID)
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
	size, checksum, err := handler.backend.Assemble(uploadID, fileID, upload.Key, partNumbers)
	if err != nil {
		err = merry.Wrap(err).WithHTTPCode(http.StatusInternalServerError)
		return
	}
	err = handler.commandHandler.CompleteUpload(ctx, filestore.CompleteUploadInput{
		UploadID:  uploadID,
		FileID:    fileID,
		Size:      size,
		Checksum:  checksum,
		UpdatedAt: time.Now().UTC().Format(time.RFC3339),
	})
	if err != nil {
		err = merrifyError(err)
		return
	}
	if deleteErr := handler.backend.DeleteParts(uploadID); deleteErr != nil {
		logx.Warnf("deleting parts of completed upload %s: %v", uploadID, deleteErr)
	}
	file, err := handler.getFile(ctx, fileID)
	if err != nil {
		return
	}
	httpx.SendJSONResponse(responseWriter, http.StatusCreated, file)
}
