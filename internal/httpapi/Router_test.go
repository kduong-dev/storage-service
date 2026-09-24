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
	"github.com/kduong-dev/storage-service/internal/fileinfostore"
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
	Convey("Given a storage service with alpha-service and beta-service clients", t, func() {
		log := eventsource.NewInMemoryLog("storage:events")
		router := httpapi.NewRouter(httpapi.NewRouterInput{
			APIKeyMiddleware: apikey.NewMiddleware(apikey.NewMiddlewareInput{
				NamespaceByKeyHash: map[string]string{
					apikey.HashAPIKey("alpha-key"): "alpha-service",
					apikey.HashAPIKey("beta-key"):  "beta-service",
				},
			}),
			FileInfoStore: fileinfostore.NewInMemoryStore(fileinfostore.NewInMemoryStoreInput{Log: log}),
			Storage:       storage.NewFileSystemStorage(storage.NewFileSystemStorageInput{Root: t.TempDir()}),
		})
		server := httptest.NewServer(router)
		defer server.Close()
		ctx := context.Background()
		alphaClient := newClient(server, "alpha-key")
		betaClient := newClient(server, "beta-key")

		Convey("When alpha-service uploads a file", func() {
			file, err := storageservice.UploadFile(ctx, alphaClient, storageservice.UploadFileInput{
				Key:         "reports/job-1/report.html",
				ContentType: "text/html",
				Body:        strings.NewReader("<h1>report</h1>"),
			})
			So(err, ShouldBeNil)

			Convey("Then the file is stored under the alpha-service namespace", func() {
				So(file.Namespace, ShouldEqual, "alpha-service")
				So(file.Key, ShouldEqual, "alpha-service/reports/job-1/report.html")
				So(file.Size, ShouldEqual, len("<h1>report</h1>"))
			})

			Convey("Then alpha-service can download it", func() {
				download, err := alphaClient.DownloadFile(ctx, file.ID)
				So(err, ShouldBeNil)
				defer download.Body.Close()
				body, err := io.ReadAll(download.Body)
				So(err, ShouldBeNil)
				So(string(body), ShouldEqual, "<h1>report</h1>")
				So(download.ContentType, ShouldEqual, "text/html")
			})

			Convey("Then beta-service cannot see it", func() {
				_, err := betaClient.DownloadFile(ctx, file.ID)
				So(errors.Is(err, storageservice.ErrFileNotFound), ShouldBeTrue)
			})
		})

		Convey("When alpha-service starts an upload", func() {
			upload, err := alphaClient.InitialiseUpload(ctx, "reports/job-2/report.html", "text/html")
			So(err, ShouldBeNil)

			Convey("Then beta-service cannot add parts to it", func() {
				_, err := betaClient.UploadPart(ctx, upload.ID, 1, strings.NewReader("intrusion"))
				So(errors.Is(err, storageservice.ErrUploadNotFound), ShouldBeTrue)
			})

			Convey("Then beta-service cannot complete it", func() {
				_, err := betaClient.CompleteUpload(ctx, upload.ID)
				So(errors.Is(err, storageservice.ErrUploadNotFound), ShouldBeTrue)
			})

			Convey("Then beta-service cannot abort it", func() {
				err := betaClient.AbortUpload(ctx, upload.ID)
				So(errors.Is(err, storageservice.ErrUploadNotFound), ShouldBeTrue)
			})

			Convey("And alpha-service aborts it", func() {
				So(alphaClient.AbortUpload(ctx, upload.ID), ShouldBeNil)

				Convey("Then no more parts can be added", func() {
					_, err := alphaClient.UploadPart(ctx, upload.ID, 1, strings.NewReader("late part"))
					So(errors.Is(err, storageservice.ErrUploadNotActive), ShouldBeTrue)
				})

				Convey("Then it cannot be completed", func() {
					_, err := alphaClient.CompleteUpload(ctx, upload.ID)
					So(errors.Is(err, storageservice.ErrUploadNotActive), ShouldBeTrue)
				})

				Convey("Then it cannot be aborted again", func() {
					err := alphaClient.AbortUpload(ctx, upload.ID)
					So(errors.Is(err, storageservice.ErrUploadNotActive), ShouldBeTrue)
				})
			})
		})

		Convey("When a client uses a key that escapes its namespace", func() {
			_, err := alphaClient.InitialiseUpload(ctx, "../beta-service/books/secret.pdf", "application/pdf")

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
