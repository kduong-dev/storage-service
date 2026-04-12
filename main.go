package main

import (
	"net/http"

	"github.com/kduong/trading-backend/cmd/storage-service/internal/filestore"
	"github.com/kduong/trading-backend/cmd/storage-service/internal/httpapi"
	"github.com/kduong/trading-backend/cmd/storage-service/internal/storage"
	"github.com/kduong/trading-backend/internal/auth"
	"github.com/kduong/trading-backend/internal/eventsource"
	"github.com/kduong/trading-backend/internal/fatal"
	"github.com/rs/cors"
)

func main() {
	logFactory, err := eventsource.LogFactoryFromEnv("STORAGE_EVENT_LOG", "INMEMORY")
	fatal.OnError(err)
	log, err := logFactory.Create("storage:events")
	fatal.OnError(err)
	commandHandler := filestore.NewCommandHandlerThreadSafeDecorator(
		filestore.NewCommandHandlerThreadSafeDecoratorInput{
			Decorated: filestore.NewEventSourcedCommandHandler(
				filestore.NewEventSourcedCommandHandlerInput{Log: log},
			),
		},
	)
	queryHandler := filestore.NewQueryHandlerThreadSafeDecorator(
		filestore.NewQueryHandlerThreadSafeDecoratorInput{
			Decorated: filestore.NewEventSourcedQueryHandler(
				filestore.NewEventSourcedQueryHandlerInput{Log: log},
			),
		},
	)
	router := httpapi.NewRouter(httpapi.NewRouterInput{
		AuthMiddleware: auth.MiddlewareFromEnv(),
		CommandHandler: commandHandler,
		QueryHandler:   queryHandler,
		Backend:        storage.FromEnv(),
	})
	c := cors.New(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "HEAD", "OPTIONS", "POST", "PUT", "DELETE"},
		AllowedHeaders:   []string{"Content-Type", "Content-Length", "Accept-Encoding", "Authorization", "Accept", "Origin", "Range"},
		ExposedHeaders:   []string{"Content-Length", "Content-Range", "Content-Disposition"},
		AllowCredentials: true,
	})
	fatal.OnError(http.ListenAndServe(":8083", c.Handler(router)))
}
