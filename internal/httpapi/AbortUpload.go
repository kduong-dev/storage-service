package httpapi

import (
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/kduong-dev/goutil/httpx"
	"github.com/kduong-dev/storage-service/internal/upload"
)

func (api *API) AbortUpload(responseWriter http.ResponseWriter, request *http.Request) {
	var err error
	defer func() {
		if err != nil {
			httpx.SendErrorResponse(responseWriter, err)
		}
	}()
	ctx := request.Context()
	vars := mux.Vars(request)
	uploadID := vars["upload_id"]
	if _, err = api.getUpload(ctx, uploadID); err != nil {
		return
	}
	err = api.uploadObjectStore.Abort(ctx, upload.AbortInput{
		UploadID:  uploadID,
		UpdatedAt: time.Now().UTC().Format(time.RFC3339),
	})
	if err = merrifiedSentinels.MerrifyOrFatal(err); err != nil {
		return
	}
	if err = api.storage.AbortUpload(ctx, uploadID); err != nil {
		return
	}
	responseWriter.WriteHeader(http.StatusNoContent)
}
