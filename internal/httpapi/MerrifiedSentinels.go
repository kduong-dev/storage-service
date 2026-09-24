package httpapi

import (
	"net/http"

	"github.com/kduong-dev/goutil/httpx"
	"github.com/kduong-dev/storage-service/internal/file"
	"github.com/kduong-dev/storage-service/internal/storage"
	"github.com/kduong-dev/storage-service/internal/upload"
)

// merrifiedSentinels are the store and storage sentinels a client can cause.
var merrifiedSentinels = httpx.MerrifiedSentinels{
	{Sentinel: upload.ErrNotFound, StatusCode: http.StatusNotFound, UserMessage: "upload not found"},
	{Sentinel: storage.ErrUploadNotFound, StatusCode: http.StatusNotFound, UserMessage: "upload not found"},
	{Sentinel: file.ErrNotFound, StatusCode: http.StatusNotFound, UserMessage: "file not found"},
	{Sentinel: file.ErrInvalidAfter, StatusCode: http.StatusBadRequest, UserMessage: "invalid cursor"},
}
