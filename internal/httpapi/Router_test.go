package httpapi_test

import (
	"context"
	"errors"
	"io"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/kduong-dev/goutil/eventsource"
	"github.com/kduong-dev/storage-service/internal/apikey"
	"github.com/kduong-dev/storage-service/internal/filestore"
	"github.com/kduong-dev/storage-service/internal/httpapi"
	"github.com/kduong-dev/storage-service/internal/storage"
	"github.com/kduong-dev/storage-service/pkg/storageservice"
	. "github.com/smartystreets/goconvey/convey"
)

func newClient(server *httptest.Server, apiKey string) storageservice.Client {
	baseURL, err := url.Parse(server.URL)
	So(err, ShouldBeNil)
	return storageservice.NewHTTPClient(storageservice.NewHTTPClientInput{
		Timeout: 5 * time.Second,
		BaseURL: *baseURL,
		APIKey:  apiKey,
	})
}

func TestRouter(t *testing.T) {
	Convey("Given a storage service with trading-core and remarkable-shelf clients", t, func() {
		log := eventsource.NewInMemoryLog("storage:events")
		router := httpapi.NewRouter(httpapi.NewRouterInput{
			APIKeyMiddleware: apikey.NewMiddleware(apikey.NewMiddlewareInput{
				NamespaceByKeyHash: map[string]string{
					apikey.HashAPIKey("trading-key"): "trading-core",
					apikey.HashAPIKey("shelf-key"):   "remarkable-shelf",
				},
			}),
			CommandHandler: filestore.NewEventSourcedCommandHandler(filestore.NewEventSourcedCommandHandlerInput{Log: log}),
			QueryHandler:   filestore.NewEventSourcedQueryHandler(filestore.NewEventSourcedQueryHandlerInput{Log: log}),
			Backend:        storage.NewInMemoryBackend(),
		})
		server := httptest.NewServer(router)
		defer server.Close()
		ctx := context.Background()
		tradingClient := newClient(server, "trading-key")
		shelfClient := newClient(server, "shelf-key")

		Convey("When trading-core uploads a file", func() {
			file, err := storageservice.UploadFile(ctx, tradingClient, storageservice.UploadFileInput{
				Key:         "reports/job-1/report.html",
				ContentType: "text/html",
				Body:        strings.NewReader("<h1>report</h1>"),
			})
			So(err, ShouldBeNil)

			Convey("Then the file is stored under the trading-core namespace", func() {
				So(file.Namespace, ShouldEqual, "trading-core")
				So(file.Key, ShouldEqual, "trading-core/reports/job-1/report.html")
				So(file.Size, ShouldEqual, len("<h1>report</h1>"))
			})

			Convey("Then trading-core can download it", func() {
				download, err := tradingClient.DownloadFile(ctx, file.ID)
				So(err, ShouldBeNil)
				defer download.Body.Close()
				body, err := io.ReadAll(download.Body)
				So(err, ShouldBeNil)
				So(string(body), ShouldEqual, "<h1>report</h1>")
				So(download.ContentType, ShouldEqual, "text/html")
			})

			Convey("Then remarkable-shelf cannot see it", func() {
				_, err := shelfClient.DownloadFile(ctx, file.ID)
				So(errors.Is(err, storageservice.ErrFileNotFound), ShouldBeTrue)
			})
		})

		Convey("When trading-core starts an upload", func() {
			upload, err := tradingClient.InitialiseUpload(ctx, "reports/job-2/report.html", "text/html")
			So(err, ShouldBeNil)

			Convey("Then remarkable-shelf cannot add parts to it", func() {
				_, err := shelfClient.UploadPart(ctx, upload.ID, 1, strings.NewReader("intrusion"))
				So(errors.Is(err, storageservice.ErrUploadNotFound), ShouldBeTrue)
			})

			Convey("Then remarkable-shelf cannot complete it", func() {
				_, err := shelfClient.CompleteUpload(ctx, upload.ID)
				So(errors.Is(err, storageservice.ErrUploadNotFound), ShouldBeTrue)
			})
		})

		Convey("When a client uses a key that escapes its namespace", func() {
			_, err := tradingClient.InitialiseUpload(ctx, "../remarkable-shelf/books/secret.pdf", "application/pdf")

			Convey("Then the upload is rejected as a bad request", func() {
				So(errors.Is(err, storageservice.ErrBadRequest), ShouldBeTrue)
			})
		})

		Convey("When a client uses an unknown API key", func() {
			_, err := newClient(server, "stolen-key").InitialiseUpload(ctx, "reports/report.html", "text/html")

			Convey("Then the request is unauthorized", func() {
				So(errors.Is(err, storageservice.ErrUnauthorized), ShouldBeTrue)
			})
		})
	})
}
