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
	object, err := handler.getUpload(request.Context(), mux.Vars(request)["upload_id"])
	if err != nil {
		return
	}
	httpx.SendJSONResponse(responseWriter, http.StatusOK, object)
}
