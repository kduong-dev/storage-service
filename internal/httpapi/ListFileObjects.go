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

func (handler *Handler) ListFileObjects(responseWriter http.ResponseWriter, request *http.Request) {
	var err error
	defer func() {
		if err != nil {
			httpx.SendErrorResponse(responseWriter, err)
		}
	}()
	ctx := request.Context()
	query := request.URL.Query()
	limit := storageservice.DefaultListFileObjectsLimit
	if rawLimit := query.Get("limit"); rawLimit != "" {
		limit, err = strconv.Atoi(rawLimit)
		if err != nil || limit < 1 || limit > storageservice.MaxListFileObjectsLimit {
			err = merry.UserErrorf("limit must be an integer between 1 and %d", storageservice.MaxListFileObjectsLimit).WithHTTPCode(http.StatusBadRequest)
			return
		}
	}
	output, err := handler.fileObjectStore.List(ctx, file.ListInput{
		KeyPrefix: apikey.GetNamespace(ctx) + "/" + query.Get("prefix"),
		After:     query.Get("cursor"),
		Limit:     limit,
	})
	if err = merrifiedSentinels.MerrifyOrFatal(err); err != nil {
		return
	}
	httpx.SendJSONResponse(responseWriter, http.StatusOK, output)
}
