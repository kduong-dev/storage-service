package httpapi

import (
	"net/http"
	"strconv"

	"github.com/ansel1/merry"
	"github.com/kduong-dev/goutil/httpx"
	"github.com/kduong-dev/storage-service/internal/apikey"
	"github.com/kduong-dev/storage-service/internal/file"
	"github.com/kduong-dev/storage-service/pkg/storageservice"
)

const (
	defaultListFilesLimit = 100
	maxListFilesLimit     = 1000
)

func (handler *Handler) ListFiles(responseWriter http.ResponseWriter, request *http.Request) {
	var err error
	defer func() {
		if err != nil {
			httpx.SendErrorResponse(responseWriter, err)
		}
	}()
	ctx := request.Context()
	query := request.URL.Query()
	limit := defaultListFilesLimit
	if rawLimit := query.Get("limit"); rawLimit != "" {
		limit, err = strconv.Atoi(rawLimit)
		if err != nil || limit < 1 || limit > maxListFilesLimit {
			err = merry.UserErrorf("limit must be an integer between 1 and %d", maxListFilesLimit).WithHTTPCode(http.StatusBadRequest)
			return
		}
	}
	output, err := handler.fileObjectStore.List(ctx, file.ListInput{
		KeyPrefix: apikey.GetNamespace(ctx) + "/" + query.Get("prefix"),
		After:     query.Get("cursor"),
		Limit:     limit,
	})
	if err = merrifyOrFatal(err); err != nil {
		return
	}
	httpx.SendJSONResponse(responseWriter, http.StatusOK, storageservice.ListFilesResponse{
		Files:      output.Objects,
		NextCursor: output.NextAfter,
	})
}
