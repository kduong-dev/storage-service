package httpapi

import (
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/kduong-dev/goutil/httpx"
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
	if _, err = handler.getActiveUpload(ctx, uploadID); err != nil {
		return
	}
	if err = handler.uploadObjectStore.Abort(ctx, uploadID, time.Now().UTC().Format(time.RFC3339)); err != nil {
		err = merrifyError(err)
		return
	}
	if err = handler.storage.AbortUpload(ctx, uploadID); err != nil {
		return
	}
	responseWriter.WriteHeader(http.StatusNoContent)
}
