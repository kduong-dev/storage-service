package upload_test

import (
	"context"
	"testing"

	"github.com/kduong-dev/goutil/eventsource"
	"github.com/kduong-dev/storage-service/internal/upload"
	. "github.com/smartystreets/goconvey/convey"
)

func TestEventSourcedObjectStore(t *testing.T) {
	Convey("Given an event-sourced object store with an initialised upload", t, func() {
		ctx := context.Background()
		log := eventsource.NewInMemoryLog("storage:uploads")
		store := upload.NewEventSourcedObjectStore(upload.NewEventSourcedObjectStoreInput{Log: log})
		So(store.Initialise(ctx, &upload.Object{
			ID:          "upload-1",
			Key:         "alpha-service/reports/report.html",
			ContentType: "text/html",
			CreatedAt:   "2026-01-01T00:00:00Z",
		}), ShouldBeNil)
		recordPart := func(part upload.Part, updatedAt string) error {
			return store.RecordPart(ctx, upload.RecordPartInput{UploadID: "upload-1", Part: part, UpdatedAt: updatedAt})
		}
		Convey("When a part is recorded twice under the same number", func() {
			So(recordPart(upload.Part{Number: 1, Size: 5, Checksum: "first"}, "2026-01-01T00:00:01Z"), ShouldBeNil)
			So(recordPart(upload.Part{Number: 1, Size: 7, Checksum: "second"}, "2026-01-01T00:00:02Z"), ShouldBeNil)
			fetched, err := store.Get(ctx, "upload-1")
			Convey("Then the later part replaces the earlier one", func() {
				So(err, ShouldBeNil)
				So(fetched.Parts, ShouldResemble, []upload.Part{{Number: 1, Size: 7, Checksum: "second"}})
				So(fetched.UpdatedAt, ShouldEqual, "2026-01-01T00:00:02Z")
			})
		})
		Convey("When the upload is completed", func() {
			So(store.Complete(ctx, upload.CompleteInput{
				UploadID:  "upload-1",
				FileID:    "file-1",
				Size:      10,
				Checksum:  "abc",
				UpdatedAt: "2026-01-01T00:00:03Z",
			}), ShouldBeNil)
			Convey("Then it is removed from the store", func() {
				_, err := store.Get(ctx, "upload-1")
				So(err, ShouldEqual, upload.ErrNotFound)
				So(recordPart(upload.Part{Number: 2}, "2026-01-01T00:00:04Z"), ShouldEqual, upload.ErrNotFound)
				So(store.Abort(ctx, upload.AbortInput{
					UploadID:  "upload-1",
					UpdatedAt: "2026-01-01T00:00:04Z",
				}), ShouldEqual, upload.ErrNotFound)
			})
			Convey("Then a store rebuilt from the event log does not have it either", func() {
				rebuilt := upload.NewEventSourcedObjectStore(upload.NewEventSourcedObjectStoreInput{Log: log})
				_, err := rebuilt.Get(ctx, "upload-1")
				So(err, ShouldEqual, upload.ErrNotFound)
			})
		})
		Convey("When the upload is aborted", func() {
			So(store.Abort(ctx, upload.AbortInput{
				UploadID:  "upload-1",
				UpdatedAt: "2026-01-01T00:00:03Z",
			}), ShouldBeNil)
			Convey("Then it is aborted and accepts no more changes", func() {
				fetched, err := store.Get(ctx, "upload-1")
				So(err, ShouldBeNil)
				So(fetched.Status, ShouldEqual, upload.StatusAborted)
				So(recordPart(upload.Part{Number: 1}, "2026-01-01T00:00:04Z"), ShouldEqual, upload.ErrNotActive)
				So(store.Complete(ctx, upload.CompleteInput{UploadID: "upload-1"}), ShouldEqual, upload.ErrNotActive)
			})
		})
		Convey("When a fetched upload is modified by the caller", func() {
			So(recordPart(upload.Part{Number: 1, Size: 5}, "2026-01-01T00:00:01Z"), ShouldBeNil)
			fetched, err := store.Get(ctx, "upload-1")
			So(err, ShouldBeNil)
			fetched.Status = upload.StatusAborted
			fetched.Parts[0].Size = 999
			Convey("Then the stored upload is unaffected", func() {
				stored, err := store.Get(ctx, "upload-1")
				So(err, ShouldBeNil)
				So(stored.Status, ShouldEqual, upload.StatusInitiated)
				So(stored.Parts[0].Size, ShouldEqual, 5)
			})
		})
		Convey("When another store is built from the same event log", func() {
			rebuilt := upload.NewEventSourcedObjectStore(upload.NewEventSourcedObjectStoreInput{Log: log})
			fetched, err := rebuilt.Get(ctx, "upload-1")
			Convey("Then it sees the uploads recorded so far", func() {
				So(err, ShouldBeNil)
				So(fetched.Key, ShouldEqual, "alpha-service/reports/report.html")
			})
		})
		Convey("When an unknown upload is changed", func() {
			err := store.RecordPart(ctx, upload.RecordPartInput{UploadID: "upload-missing", Part: upload.Part{Number: 1}})
			Convey("Then it reports the upload as not found", func() {
				So(err, ShouldEqual, upload.ErrNotFound)
			})
		})
	})
}
