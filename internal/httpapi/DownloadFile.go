package httpapi

import (
	"net/http"
	"path/filepath"
	"time"

	"github.com/gorilla/mux"
	"github.com/kduong-dev/goutil/httpx"
)

func (handler *Handler) DownloadFile(responseWriter http.ResponseWriter, request *http.Request) {
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
	readSeekCloser, err := handler.storage.OpenFile(object.ID)
	if err != nil {
		return
	}
	defer readSeekCloser.Close()
	filename := filepath.Base(object.Key)
	responseWriter.Header().Set("Content-Type", object.ContentType)
	responseWriter.Header().Set("Content-Disposition", `attachment; filename="`+filename+`"`)
	http.ServeContent(responseWriter, request, filename, time.Time{}, readSeekCloser)
}
