package main

import (
	"net/http"

	"github.com/kduong-dev/goutil/config"
	"github.com/kduong-dev/goutil/eventsource"
	"github.com/kduong-dev/goutil/fatal"
	"github.com/kduong-dev/goutil/logx"
	"github.com/kduong-dev/storage-service/internal/apikey"
	"github.com/kduong-dev/storage-service/internal/fileinfostore"
	"github.com/kduong-dev/storage-service/internal/httpapi"
	"github.com/kduong-dev/storage-service/internal/storage"
)

func main() {
	logFactory, err := eventsource.LogFactoryFromEnv("STORAGE_EVENT_LOG", "INMEMORY")
	fatal.OnError(err)
	log, err := logFactory.Create("storage:events")
	fatal.OnError(err)
	legacyNamespace := config.EnvString("STORAGE_LEGACY_NAMESPACE", "")
	commandHandler := fileinfostore.NewCommandHandlerThreadSafeDecorator(
		fileinfostore.NewCommandHandlerThreadSafeDecoratorInput{
			Decorated: fileinfostore.NewEventSourcedCommandHandler(fileinfostore.NewEventSourcedCommandHandlerInput{
				Log:             log,
				LegacyNamespace: legacyNamespace,
			}),
		},
	)
	queryHandler := fileinfostore.NewQueryHandlerThreadSafeDecorator(
		fileinfostore.NewQueryHandlerThreadSafeDecoratorInput{
			Decorated: fileinfostore.NewEventSourcedQueryHandler(fileinfostore.NewEventSourcedQueryHandlerInput{
				Log:             log,
				LegacyNamespace: legacyNamespace,
			}),
		},
	)
	router := httpapi.NewRouter(httpapi.NewRouterInput{
		APIKeyMiddleware: apikey.MiddlewareFromEnv(),
		CommandHandler:   commandHandler,
		QueryHandler:     queryHandler,
		Storage:          storage.FromEnv(),
	})
	address := ":" + config.EnvString("PORT", "8083")
	logx.Noticef("storage-service listening on %s", address)
	fatal.OnError(http.ListenAndServe(address, router))
}
