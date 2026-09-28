package httpapi

import (
	"net/http"

	"github.com/gorilla/mux"
)

func (api *API) DeleteFile(responseWriter http.ResponseWriter, request *http.Request) {
	var err error
	defer func() {
		if err != nil {
			merrifiedSentinels.SendErrorResponse(responseWriter, err)
		}
	}()
	ctx := request.Context()
	vars := mux.Vars(request)
	fileID := vars["file_id"]
	if _, err = api.getFile(ctx, fileID); err != nil {
		return
	}
	err = api.fileObjectStore.Delete(ctx, fileID)
	if err != nil {
		return
	}
	if err = api.storage.DeleteFile(ctx, fileID); err != nil {
		return
	}
	responseWriter.WriteHeader(http.StatusNoContent)
}
