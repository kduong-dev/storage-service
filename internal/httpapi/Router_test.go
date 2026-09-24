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
	"github.com/kduong-dev/storage-service/internal/file"
	"github.com/kduong-dev/storage-service/internal/httpapi"
	"github.com/kduong-dev/storage-service/internal/storage"
	"github.com/kduong-dev/storage-service/internal/upload"
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
		router := httpapi.NewRouter(httpapi.NewRouterInput{
			APIKeyMiddleware: apikey.NewMiddleware(apikey.NewMiddlewareInput{
				NamespaceByKeyHash: map[string]string{
					apikey.HashAPIKey("alpha-key"):  "alpha-service",
					apikey.HashAPIKey("beta-key"):   "beta-service",
					apikey.HashAPIKey("prefix-key"): "alpha",
				},
			}),
			UploadObjectStore: upload.NewObjectStoreThreadSafeDecorator(upload.NewObjectStoreThreadSafeDecoratorInput{
				Decorated: upload.NewEventSourcedObjectStore(upload.NewEventSourcedObjectStoreInput{
					Log: eventsource.NewInMemoryLog("storage:uploads"),
				}),
			}),
			FileObjectStore: file.NewObjectStoreThreadSafeDecorator(file.NewObjectStoreThreadSafeDecoratorInput{
				Decorated: file.NewEventSourcedObjectStore(file.NewEventSourcedObjectStoreInput{
					Log: eventsource.NewInMemoryLog("storage:files"),
				}),
			}),
			Storage: storage.NewFileSystemStorage(storage.NewFileSystemStorageInput{Root: t.TempDir()}),
		})
		server := httptest.NewServer(router)
		defer server.Close()
		ctx := context.Background()
		alphaClient := newClient(server, "alpha-key")
		betaClient := newClient(server, "beta-key")
		Convey("When alpha-service uploads a file", func() {
			uploadedFile, err := storageservice.UploadFile(ctx, alphaClient, storageservice.UploadFileInput{
				Key:         "reports/job-1/report.html",
				ContentType: "text/html",
				Body:        strings.NewReader("<h1>report</h1>"),
			})
			So(err, ShouldBeNil)
			Convey("Then the file is stored under the alpha-service namespace", func() {
				So(uploadedFile.Key, ShouldEqual, "alpha-service/reports/job-1/report.html")
				So(uploadedFile.Size, ShouldEqual, len("<h1>report</h1>"))
			})
			Convey("Then alpha-service can download it", func() {
				download, err := alphaClient.DownloadFile(ctx, uploadedFile.ID)
				So(err, ShouldBeNil)
				defer download.Body.Close()
				body, err := io.ReadAll(download.Body)
				So(err, ShouldBeNil)
				So(string(body), ShouldEqual, "<h1>report</h1>")
				So(download.ContentType, ShouldEqual, "text/html")
			})
			Convey("Then beta-service cannot see it", func() {
				_, err := betaClient.DownloadFile(ctx, uploadedFile.ID)
				So(errors.Is(err, storageservice.ErrFileNotFound), ShouldBeTrue)
			})
			Convey("Then a namespace that is a prefix of alpha-service cannot see it", func() {
				_, err := newClient(server, "prefix-key").DownloadFile(ctx, uploadedFile.ID)
				So(errors.Is(err, storageservice.ErrFileNotFound), ShouldBeTrue)
			})
		})
		Convey("When alpha-service starts an upload", func() {
			startedUpload, err := alphaClient.InitialiseUpload(ctx, "reports/job-2/report.html", "text/html")
			So(err, ShouldBeNil)
			Convey("Then beta-service cannot add parts to it", func() {
				_, err := betaClient.UploadPart(ctx, startedUpload.ID, 1, strings.NewReader("intrusion"))
				So(errors.Is(err, storageservice.ErrUploadNotFound), ShouldBeTrue)
			})
			Convey("Then beta-service cannot complete it", func() {
				_, err := betaClient.CompleteUpload(ctx, startedUpload.ID)
				So(errors.Is(err, storageservice.ErrUploadNotFound), ShouldBeTrue)
			})
			Convey("Then a part over 5 MB is rejected as a bad request", func() {
				_, err := alphaClient.UploadPart(ctx, startedUpload.ID, 1, strings.NewReader(strings.Repeat("a", 5*1024*1024+1)))
				So(errors.Is(err, storageservice.ErrBadRequest), ShouldBeTrue)
			})
			Convey("Then beta-service cannot abort it", func() {
				err := betaClient.AbortUpload(ctx, startedUpload.ID)
				So(errors.Is(err, storageservice.ErrUploadNotFound), ShouldBeTrue)
			})
			Convey("And alpha-service aborts it", func() {
				So(alphaClient.AbortUpload(ctx, startedUpload.ID), ShouldBeNil)
				Convey("Then no more parts can be added", func() {
					_, err := alphaClient.UploadPart(ctx, startedUpload.ID, 1, strings.NewReader("late part"))
					So(errors.Is(err, storageservice.ErrUploadNotFound), ShouldBeTrue)
				})
				Convey("Then it cannot be completed", func() {
					_, err := alphaClient.CompleteUpload(ctx, startedUpload.ID)
					So(errors.Is(err, storageservice.ErrUploadNotFound), ShouldBeTrue)
				})
				Convey("Then it cannot be aborted again", func() {
					err := alphaClient.AbortUpload(ctx, startedUpload.ID)
					So(errors.Is(err, storageservice.ErrUploadNotFound), ShouldBeTrue)
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
