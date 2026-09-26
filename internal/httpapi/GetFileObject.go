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
	object, err := handler.getFile(request.Context(), mux.Vars(request)["file_id"])
	if err != nil {
		return
	}
	httpx.SendJSONResponse(responseWriter, http.StatusOK, object)
}
