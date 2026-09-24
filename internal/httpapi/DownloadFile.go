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
	fileInfo, err := handler.getFileInfo(ctx, fileID)
	if err != nil {
		return
	}
	readSeekCloser, err := handler.storage.OpenFile(fileInfo.Key)
	if err != nil {
		return
	}
	defer readSeekCloser.Close()
	filename := filepath.Base(fileInfo.Key)
	responseWriter.Header().Set("Content-Type", fileInfo.ContentType)
	responseWriter.Header().Set("Content-Disposition", `attachment; filename="`+filename+`"`)
	http.ServeContent(responseWriter, request, filename, time.Time{}, readSeekCloser)
}
