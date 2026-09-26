package httpapi

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/kduong-dev/goutil/httpx"
)

func (handler *Handler) GetFileObject(responseWriter http.ResponseWriter, request *http.Request) {
	var err error
	defer func() {
		if err != nil {
			httpx.SendErrorResponse(responseWriter, err)
		}
	}()
	ctx := request.Context()
	vars := mux.Vars(request)
	fileID := vars["file_id"]
	object, err := handler.getFile(ctx, fileID)
	if err != nil {
		return
	}
	httpx.SendJSONResponse(responseWriter, http.StatusOK, object)
}
