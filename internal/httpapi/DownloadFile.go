package httpapi

import (
	"net/http"
	"path/filepath"
	"time"

	"github.com/gorilla/mux"
	"github.com/kduong/trading-backend/internal/authz"
	"github.com/kduong/trading-backend/internal/httpx"
)

var zeroTime = time.Time{}

func (handler *Handler) DownloadFile(responseWriter http.ResponseWriter, request *http.Request) {
	var err error
	defer func() {
		if err != nil {
			httpx.SendErrorResponse(responseWriter, err)
		}
	}()
	ctx := request.Context()
	if err = authz.RequireScope(ctx, authz.ScopeFilesRead); err != nil {
		return
	}
	vars := mux.Vars(request)
	fileID := vars["file_id"]
	file, err := handler.queryHandler.GetFile(ctx, fileID)
	if err != nil {
		err = merrifyError(err)
		return
	}
	readSeekCloser, err := handler.backend.Open(file.Key)
	if err != nil {
		err = merrifyError(err)
		return
	}
	defer readSeekCloser.Close()
	filename := filepath.Base(file.Key)
	responseWriter.Header().Set("Content-Type", file.ContentType)
	responseWriter.Header().Set("Content-Disposition", `attachment; filename="`+filename+`"`)
	http.ServeContent(responseWriter, request, filename, zeroTime, readSeekCloser)
}
