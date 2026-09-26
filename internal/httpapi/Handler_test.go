package httpapi_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/ansel1/merry"
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
		BaseURL: baseURL,
		APIKey:  apiKey,
	})
}

func TestHandler(t *testing.T) {
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
				download, err := alphaClient.DownloadFile(ctx, storageservice.DownloadFileInput{FileID: uploadedFile.ID})
				So(err, ShouldBeNil)
				defer download.Body.Close()
				body, err := io.ReadAll(download.Body)
				So(err, ShouldBeNil)
				So(string(body), ShouldEqual, "<h1>report</h1>")
				So(download.ContentType, ShouldEqual, "text/html")
			})
			Convey("Then alpha-service can download a byte range of it", func() {
				download, err := alphaClient.DownloadFile(ctx, storageservice.DownloadFileInput{FileID: uploadedFile.ID, Range: "bytes=4-9"})
				So(err, ShouldBeNil)
				defer download.Body.Close()
				body, err := io.ReadAll(download.Body)
				So(err, ShouldBeNil)
				So(string(body), ShouldEqual, "report")
				So(download.ContentRange, ShouldEqual, "bytes 4-9/15")
			})
			Convey("Then a range past the end of the file is not satisfiable", func() {
				_, err := alphaClient.DownloadFile(ctx, storageservice.DownloadFileInput{FileID: uploadedFile.ID, Range: "bytes=100-"})
				So(errors.Is(err, storageservice.ErrRangeNotSatisfiable), ShouldBeTrue)
				So(merry.HTTPCode(err), ShouldEqual, http.StatusRequestedRangeNotSatisfiable)
			})
			Convey("Then alpha-service can read its metadata", func() {
				metadata, err := alphaClient.GetFileObject(ctx, storageservice.GetFileObjectInput{FileID: uploadedFile.ID})
				So(err, ShouldBeNil)
				So(metadata, ShouldResemble, uploadedFile)
			})
			Convey("Then beta-service cannot read its metadata", func() {
				_, err := betaClient.GetFileObject(ctx, storageservice.GetFileObjectInput{FileID: uploadedFile.ID})
				So(errors.Is(err, storageservice.ErrFileNotFound), ShouldBeTrue)
			})
			Convey("Then beta-service cannot see it", func() {
				_, err := betaClient.DownloadFile(ctx, storageservice.DownloadFileInput{FileID: uploadedFile.ID})
				So(errors.Is(err, storageservice.ErrFileNotFound), ShouldBeTrue)
			})
			Convey("Then a namespace that is a prefix of alpha-service cannot see it", func() {
				_, err := newClient(server, "prefix-key").DownloadFile(ctx, storageservice.DownloadFileInput{FileID: uploadedFile.ID})
				So(errors.Is(err, storageservice.ErrFileNotFound), ShouldBeTrue)
			})
			Convey("Then beta-service cannot delete it", func() {
				err := betaClient.DeleteFile(ctx, storageservice.DeleteFileInput{FileID: uploadedFile.ID})
				So(errors.Is(err, storageservice.ErrFileNotFound), ShouldBeTrue)
				_, err = alphaClient.DownloadFile(ctx, storageservice.DownloadFileInput{FileID: uploadedFile.ID})
				So(err, ShouldBeNil)
			})
			Convey("And alpha-service deletes it", func() {
				So(alphaClient.DeleteFile(ctx, storageservice.DeleteFileInput{FileID: uploadedFile.ID}), ShouldBeNil)
				Convey("Then it can no longer be downloaded", func() {
					_, err := alphaClient.DownloadFile(ctx, storageservice.DownloadFileInput{FileID: uploadedFile.ID})
					So(errors.Is(err, storageservice.ErrFileNotFound), ShouldBeTrue)
				})
				Convey("Then its metadata can no longer be read", func() {
					_, err := alphaClient.GetFileObject(ctx, storageservice.GetFileObjectInput{FileID: uploadedFile.ID})
					So(errors.Is(err, storageservice.ErrFileNotFound), ShouldBeTrue)
				})
				Convey("Then it is no longer listed", func() {
					response, err := alphaClient.ListFileObjects(ctx, storageservice.ListFileObjectsInput{})
					So(err, ShouldBeNil)
					So(response.Files, ShouldBeEmpty)
				})
				Convey("Then it cannot be deleted again", func() {
					err := alphaClient.DeleteFile(ctx, storageservice.DeleteFileInput{FileID: uploadedFile.ID})
					So(errors.Is(err, storageservice.ErrFileNotFound), ShouldBeTrue)
				})
			})
			Convey("And alpha-service uploads to the same key again", func() {
				secondFile, err := storageservice.UploadFile(ctx, alphaClient, storageservice.UploadFileInput{
					Key:         "reports/job-1/report.html",
					ContentType: "text/html",
					Body:        strings.NewReader("<h1>second</h1>"),
				})
				So(err, ShouldBeNil)
				downloadBody := func(fileID string) string {
					download, err := alphaClient.DownloadFile(ctx, storageservice.DownloadFileInput{FileID: fileID})
					So(err, ShouldBeNil)
					defer download.Body.Close()
					body, err := io.ReadAll(download.Body)
					So(err, ShouldBeNil)
					return string(body)
				}
				Convey("Then each file keeps its own content", func() {
					So(downloadBody(uploadedFile.ID), ShouldEqual, "<h1>report</h1>")
					So(downloadBody(secondFile.ID), ShouldEqual, "<h1>second</h1>")
				})
				Convey("Then both are listed, oldest first", func() {
					response, err := alphaClient.ListFileObjects(ctx, storageservice.ListFileObjectsInput{})
					So(err, ShouldBeNil)
					So(len(response.Files), ShouldEqual, 2)
					So(response.Files[0].ID, ShouldEqual, uploadedFile.ID)
					So(response.Files[1].ID, ShouldEqual, secondFile.ID)
				})
				Convey("And the first file is deleted", func() {
					So(alphaClient.DeleteFile(ctx, storageservice.DeleteFileInput{FileID: uploadedFile.ID}), ShouldBeNil)
					Convey("Then the second file can still be downloaded", func() {
						So(downloadBody(secondFile.ID), ShouldEqual, "<h1>second</h1>")
					})
				})
			})
		})
		Convey("When alpha-service starts an upload", func() {
			startedUpload, err := alphaClient.InitialiseUpload(ctx, storageservice.InitialiseUploadInput{Key: "reports/job-2/report.html", ContentType: "text/html"})
			So(err, ShouldBeNil)
			Convey("And alpha-service uploads a part", func() {
				_, err := alphaClient.UploadPart(ctx, storageservice.UploadPartInput{UploadID: startedUpload.ID, PartNumber: 1, Body: strings.NewReader("first part")})
				So(err, ShouldBeNil)
				Convey("Then alpha-service can see the part it received", func() {
					uploadObject, err := alphaClient.GetUploadObject(ctx, storageservice.GetUploadObjectInput{UploadID: startedUpload.ID})
					So(err, ShouldBeNil)
					So(uploadObject.Key, ShouldEqual, "alpha-service/reports/job-2/report.html")
					So(len(uploadObject.Parts), ShouldEqual, 1)
					So(uploadObject.Parts[0].PartNumber, ShouldEqual, 1)
					So(uploadObject.Parts[0].Size, ShouldEqual, len("first part"))
				})
				Convey("Then beta-service cannot see it", func() {
					_, err := betaClient.GetUploadObject(ctx, storageservice.GetUploadObjectInput{UploadID: startedUpload.ID})
					So(errors.Is(err, storageservice.ErrUploadNotFound), ShouldBeTrue)
				})
				Convey("And alpha-service completes it", func() {
					_, err := alphaClient.CompleteUpload(ctx, storageservice.CompleteUploadInput{UploadID: startedUpload.ID})
					So(err, ShouldBeNil)
					Convey("Then the upload can no longer be fetched", func() {
						_, err := alphaClient.GetUploadObject(ctx, storageservice.GetUploadObjectInput{UploadID: startedUpload.ID})
						So(errors.Is(err, storageservice.ErrUploadNotFound), ShouldBeTrue)
					})
				})
			})
			Convey("Then beta-service cannot add parts to it", func() {
				_, err := betaClient.UploadPart(ctx, storageservice.UploadPartInput{UploadID: startedUpload.ID, PartNumber: 1, Body: strings.NewReader("intrusion")})
				So(errors.Is(err, storageservice.ErrUploadNotFound), ShouldBeTrue)
				So(merry.HTTPCode(err), ShouldEqual, http.StatusNotFound)
				So(merry.UserMessage(err), ShouldEqual, "upload not found")
			})
			Convey("Then beta-service cannot complete it", func() {
				_, err := betaClient.CompleteUpload(ctx, storageservice.CompleteUploadInput{UploadID: startedUpload.ID})
				So(errors.Is(err, storageservice.ErrUploadNotFound), ShouldBeTrue)
			})
			Convey("Then a part over 5 MB is rejected as a bad request", func() {
				_, err := alphaClient.UploadPart(ctx, storageservice.UploadPartInput{UploadID: startedUpload.ID, PartNumber: 1, Body: strings.NewReader(strings.Repeat("a", 5*1024*1024+1))})
				So(errors.Is(err, storageservice.ErrBadRequest), ShouldBeTrue)
				So(merry.HTTPCode(err), ShouldEqual, http.StatusRequestEntityTooLarge)
				So(merry.UserMessage(err), ShouldEqual, "part exceeds the 5 MB size limit")
			})
			Convey("Then beta-service cannot abort it", func() {
				err := betaClient.AbortUpload(ctx, storageservice.AbortUploadInput{UploadID: startedUpload.ID})
				So(errors.Is(err, storageservice.ErrUploadNotFound), ShouldBeTrue)
			})
			Convey("And alpha-service aborts it", func() {
				So(alphaClient.AbortUpload(ctx, storageservice.AbortUploadInput{UploadID: startedUpload.ID}), ShouldBeNil)
				Convey("Then no more parts can be added", func() {
					_, err := alphaClient.UploadPart(ctx, storageservice.UploadPartInput{UploadID: startedUpload.ID, PartNumber: 1, Body: strings.NewReader("late part")})
					So(errors.Is(err, storageservice.ErrUploadNotFound), ShouldBeTrue)
				})
				Convey("Then it cannot be completed", func() {
					_, err := alphaClient.CompleteUpload(ctx, storageservice.CompleteUploadInput{UploadID: startedUpload.ID})
					So(errors.Is(err, storageservice.ErrUploadNotFound), ShouldBeTrue)
				})
				Convey("Then it cannot be aborted again", func() {
					err := alphaClient.AbortUpload(ctx, storageservice.AbortUploadInput{UploadID: startedUpload.ID})
					So(errors.Is(err, storageservice.ErrUploadNotFound), ShouldBeTrue)
				})
			})
		})
		Convey("When a client uses a key that escapes its namespace", func() {
			_, err := alphaClient.InitialiseUpload(ctx, storageservice.InitialiseUploadInput{Key: "../beta-service/books/secret.pdf", ContentType: "application/pdf"})
			Convey("Then the upload is rejected as a bad request", func() {
				So(errors.Is(err, storageservice.ErrBadRequest), ShouldBeTrue)
			})
		})
		Convey("When alpha-service and beta-service have uploaded files", func() {
			for _, key := range []string{"reports/b.html", "reports/a.html", "images/logo.png"} {
				_, err := storageservice.UploadFile(ctx, alphaClient, storageservice.UploadFileInput{Key: key, ContentType: "text/plain", Body: strings.NewReader(key)})
				So(err, ShouldBeNil)
			}
			_, err := storageservice.UploadFile(ctx, betaClient, storageservice.UploadFileInput{Key: "reports/a.html", ContentType: "text/plain", Body: strings.NewReader("beta")})
			So(err, ShouldBeNil)
			listKeys := func(response *storageservice.ListFileObjectsOutput) []string {
				keys := make([]string, len(response.Files))
				for index, listedFile := range response.Files {
					keys[index] = listedFile.Key
				}
				return keys
			}
			Convey("Then alpha-service pages through only its own files in key order", func() {
				firstPage, err := alphaClient.ListFileObjects(ctx, storageservice.ListFileObjectsInput{Limit: 2})
				So(err, ShouldBeNil)
				So(listKeys(firstPage), ShouldResemble, []string{"alpha-service/images/logo.png", "alpha-service/reports/a.html"})
				So(firstPage.NextCursor, ShouldNotBeEmpty)
				secondPage, err := alphaClient.ListFileObjects(ctx, storageservice.ListFileObjectsInput{Cursor: firstPage.NextCursor, Limit: 2})
				So(err, ShouldBeNil)
				So(listKeys(secondPage), ShouldResemble, []string{"alpha-service/reports/b.html"})
				So(secondPage.NextCursor, ShouldBeEmpty)
			})
			Convey("Then alpha-service can filter by prefix", func() {
				response, err := alphaClient.ListFileObjects(ctx, storageservice.ListFileObjectsInput{Prefix: "reports/"})
				So(err, ShouldBeNil)
				So(listKeys(response), ShouldResemble, []string{"alpha-service/reports/a.html", "alpha-service/reports/b.html"})
			})
			Convey("Then beta-service cannot continue from alpha-service's cursor", func() {
				firstPage, err := alphaClient.ListFileObjects(ctx, storageservice.ListFileObjectsInput{Limit: 1})
				So(err, ShouldBeNil)
				_, err = betaClient.ListFileObjects(ctx, storageservice.ListFileObjectsInput{Cursor: firstPage.NextCursor})
				So(errors.Is(err, storageservice.ErrBadRequest), ShouldBeTrue)
			})
			Convey("Then a limit above 1000 is rejected as a bad request", func() {
				_, err := alphaClient.ListFileObjects(ctx, storageservice.ListFileObjectsInput{Limit: 1001})
				So(errors.Is(err, storageservice.ErrBadRequest), ShouldBeTrue)
			})
		})
		Convey("When a client uses an unknown API key", func() {
			_, err := newClient(server, "stolen-key").InitialiseUpload(ctx, storageservice.InitialiseUploadInput{Key: "reports/report.html", ContentType: "text/html"})
			Convey("Then the request is unauthorized", func() {
				So(errors.Is(err, storageservice.ErrUnauthorized), ShouldBeTrue)
			})
		})
	})
}
