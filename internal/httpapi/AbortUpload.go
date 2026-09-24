package httpapi

import (
	"errors"
	"net/http"
	"time"

	"github.com/ansel1/merry"
	"github.com/gorilla/mux"
	"github.com/kduong-dev/goutil/fatal"
	"github.com/kduong-dev/goutil/httpx"
	"github.com/kduong-dev/storage-service/internal/upload"
)

func (handler *Handler) AbortUpload(responseWriter http.ResponseWriter, request *http.Request) {
	var err error
	defer func() {
		if err != nil {
			httpx.SendErrorResponse(responseWriter, err)
		}
	}()
	ctx := request.Context()
	uploadID := mux.Vars(request)["upload_id"]
	if _, err = handler.getUpload(ctx, uploadID); err != nil {
		return
	}
	err = handler.uploadObjectStore.Abort(ctx, upload.AbortInput{
		UploadID:  uploadID,
		UpdatedAt: time.Now().UTC().Format(time.RFC3339),
	})
	if errors.Is(err, upload.ErrNotFound) {
		err = merry.Wrap(err).WithHTTPCode(http.StatusNotFound).WithUserMessage("upload not found")
		return
	}
	fatal.OnError(err)
	if err = handler.storage.AbortUpload(ctx, uploadID); err != nil {
		return
	}
	responseWriter.WriteHeader(http.StatusNoContent)
}
