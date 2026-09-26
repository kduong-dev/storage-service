package httpapi

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/kduong-dev/goutil/httpx"
)

func (handler *Handler) GetUploadObject(responseWriter http.ResponseWriter, request *http.Request) {
	var err error
	defer func() {
		if err != nil {
			httpx.SendErrorResponse(responseWriter, err)
		}
	}()
	ctx := request.Context()
	vars := mux.Vars(request)
	uploadID := vars["upload_id"]
	object, err := handler.getUpload(ctx, uploadID)
	if err != nil {
		return
	}
	httpx.SendJSONResponse(responseWriter, http.StatusOK, object)
}
