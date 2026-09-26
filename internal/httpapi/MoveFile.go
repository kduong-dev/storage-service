package httpapi

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/kduong-dev/goutil/httpx"
	"github.com/kduong-dev/storage-service/internal/apikey"
	"github.com/kduong-dev/storage-service/internal/file"
	"github.com/kduong-dev/storage-service/pkg/storageservice"
)

func (handler *Handler) MoveFile(responseWriter http.ResponseWriter, request *http.Request) {
	var err error
	defer func() {
		if err != nil {
			httpx.SendErrorResponse(responseWriter, err)
		}
	}()
	ctx := request.Context()
	vars := mux.Vars(request)
	fileID := vars["file_id"]
	requestBody, err := httpx.DecodeJSONBody[storageservice.MoveFileRequestBody](request)
	if err != nil {
		return
	}
	if err = validateKey(requestBody.Key); err != nil {
		return
	}
	object, err := handler.getFile(ctx, fileID)
	if err != nil {
		return
	}
	object.Key = apikey.GetNamespace(ctx) + "/" + requestBody.Key
	err = handler.fileObjectStore.Move(ctx, file.MoveInput{FileID: object.ID, Key: object.Key})
	if err = merrifiedSentinels.MerrifyOrFatal(err); err != nil {
		return
	}
	httpx.SendJSONResponse(responseWriter, http.StatusOK, object)
}
