package main

import (
	"net/http"

	"github.com/kduong-dev/goutil/config"
	"github.com/kduong-dev/goutil/eventsource"
	"github.com/kduong-dev/goutil/fatal"
	"github.com/kduong-dev/goutil/logx"
	"github.com/kduong-dev/storage-service/internal/apikey"
	"github.com/kduong-dev/storage-service/internal/file"
	"github.com/kduong-dev/storage-service/internal/httpapi"
	"github.com/kduong-dev/storage-service/internal/storage"
	"github.com/kduong-dev/storage-service/internal/upload"
)

func main() {
	logFactory, err := eventsource.LogFactoryFromEnv("STORAGE_EVENT_LOG", "INMEMORY")
	fatal.OnError(err)
	uploadLog, err := logFactory.Create("storage:uploads")
	fatal.OnError(err)
	fileLog, err := logFactory.Create("storage:files")
	fatal.OnError(err)
	router := httpapi.NewRouter(httpapi.NewRouterInput{
		APIKeyMiddleware: apikey.MiddlewareFromEnv(),
		UploadObjectStore: upload.NewObjectStoreThreadSafeDecorator(upload.NewObjectStoreThreadSafeDecoratorInput{
			Decorated: upload.NewEventSourcedObjectStore(upload.NewEventSourcedObjectStoreInput{Log: uploadLog}),
		}),
		FileObjectStore: file.NewObjectStoreThreadSafeDecorator(file.NewObjectStoreThreadSafeDecoratorInput{
			Decorated: file.NewEventSourcedObjectStore(file.NewEventSourcedObjectStoreInput{Log: fileLog}),
		}),
		Storage: storage.FromEnv(),
	})
	address := ":" + config.EnvString("PORT", "8083")
	logx.Noticef("storage-service listening on %s", address)
	fatal.OnError(http.ListenAndServe(address, router))
}
