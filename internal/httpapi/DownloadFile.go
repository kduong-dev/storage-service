package httpapi

import (
	"net/http"
	"path/filepath"
	"time"

	"github.com/ansel1/merry"
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
	file, err := handler.getFile(ctx, mux.Vars(request)["file_id"])
	if err != nil {
		return
	}
	readSeekCloser, err := handler.backend.Open(file.Key)
	if err != nil {
		err = merry.Wrap(err).WithHTTPCode(http.StatusInternalServerError)
		return
	}
	defer readSeekCloser.Close()
	filename := filepath.Base(file.Key)
	responseWriter.Header().Set("Content-Type", file.ContentType)
	responseWriter.Header().Set("Content-Disposition", `attachment; filename="`+filename+`"`)
	http.ServeContent(responseWriter, request, filename, time.Time{}, readSeekCloser)
}
