package httpapi

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/kduong-dev/goutil/httpx"
)

func (handler *Handler) DeleteFile(responseWriter http.ResponseWriter, request *http.Request) {
	var err error
	defer func() {
		if err != nil {
			httpx.SendErrorResponse(responseWriter, err)
		}
	}()
	ctx := request.Context()
	vars := mux.Vars(request)
	fileID := vars["file_id"]
	if _, err = handler.getFile(ctx, fileID); err != nil {
		return
	}
	err = handler.fileObjectStore.Delete(ctx, fileID)
	if err = merrifiedSentinels.MerrifyOrFatal(err); err != nil {
		return
	}
	if err = handler.storage.DeleteFile(ctx, fileID); err != nil {
		return
	}
	responseWriter.WriteHeader(http.StatusNoContent)
}
