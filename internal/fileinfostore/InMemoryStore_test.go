package fileinfostore_test

import (
	"context"
	"testing"

	"github.com/kduong-dev/goutil/eventsource"
	"github.com/kduong-dev/storage-service/internal/fileinfostore"
	"github.com/kduong-dev/storage-service/internal/upload"
	. "github.com/smartystreets/goconvey/convey"
)

func TestInMemoryStore(t *testing.T) {
	Convey("Given an in-memory store backed by an empty event log", t, func() {
		ctx := context.Background()
		log := eventsource.NewInMemoryLog("storage:events")
		store := fileinfostore.NewInMemoryStore(fileinfostore.NewInMemoryStoreInput{Log: log})
		So(store.InitialiseUpload(ctx, &upload.Object{
			ID:          "upload-1",
			Namespace:   "trading-core",
			Key:         "trading-core/reports/report.html",
			ContentType: "text/html",
			CreatedAt:   "2026-01-01T00:00:00Z",
		}), ShouldBeNil)

		Convey("When a part is recorded twice under the same number", func() {
			So(store.RecordPart(ctx, "upload-1", upload.Part{Number: 1, Size: 5, Checksum: "first"}, "2026-01-01T00:00:01Z"), ShouldBeNil)
			So(store.RecordPart(ctx, "upload-1", upload.Part{Number: 1, Size: 7, Checksum: "second"}, "2026-01-01T00:00:02Z"), ShouldBeNil)
			object, err := store.GetUpload(ctx, "upload-1")

			Convey("Then the later part replaces the earlier one", func() {
				So(err, ShouldBeNil)
				So(object.Parts, ShouldResemble, []upload.Part{{Number: 1, Size: 7, Checksum: "second"}})
				So(object.UpdatedAt, ShouldEqual, "2026-01-01T00:00:02Z")
			})
		})

		Convey("When the upload is completed", func() {
			So(store.CompleteUpload(ctx, fileinfostore.CompleteUploadInput{
				UploadID:  "upload-1",
				FileID:    "file-1",
				Size:      10,
				Checksum:  "abc",
				UpdatedAt: "2026-01-01T00:00:03Z",
			}), ShouldBeNil)

			Convey("Then the file info carries the upload's namespace, key and content type", func() {
				fileInfo, err := store.GetFileInfo(ctx, "file-1")
				So(err, ShouldBeNil)
				So(fileInfo.Namespace, ShouldEqual, "trading-core")
				So(fileInfo.Key, ShouldEqual, "trading-core/reports/report.html")
				So(fileInfo.ContentType, ShouldEqual, "text/html")
				So(fileInfo.Size, ShouldEqual, 10)
			})

			Convey("Then the upload is completed and accepts no more changes", func() {
				object, err := store.GetUpload(ctx, "upload-1")
				So(err, ShouldBeNil)
				So(object.Status, ShouldEqual, upload.StatusCompleted)
				So(store.AbortUpload(ctx, "upload-1", "2026-01-01T00:00:04Z"), ShouldEqual, fileinfostore.ErrUploadNotActive)
			})
		})

		Convey("When the upload is aborted", func() {
			So(store.AbortUpload(ctx, "upload-1", "2026-01-01T00:00:03Z"), ShouldBeNil)

			Convey("Then no more parts can be recorded", func() {
				err := store.RecordPart(ctx, "upload-1", upload.Part{Number: 1}, "2026-01-01T00:00:04Z")
				So(err, ShouldEqual, fileinfostore.ErrUploadNotActive)
			})
		})

		Convey("When a returned upload is modified by the caller", func() {
			So(store.RecordPart(ctx, "upload-1", upload.Part{Number: 1, Size: 5}, "2026-01-01T00:00:01Z"), ShouldBeNil)
			object, err := store.GetUpload(ctx, "upload-1")
			So(err, ShouldBeNil)
			object.Status = upload.StatusAborted
			object.Parts[0].Size = 999

			Convey("Then the stored upload is unaffected", func() {
				stored, err := store.GetUpload(ctx, "upload-1")
				So(err, ShouldBeNil)
				So(stored.Status, ShouldEqual, upload.StatusInitiated)
				So(stored.Parts[0].Size, ShouldEqual, 5)
			})
		})

		Convey("When another store is built from the same event log", func() {
			rebuilt := fileinfostore.NewInMemoryStore(fileinfostore.NewInMemoryStoreInput{Log: log})
			object, err := rebuilt.GetUpload(ctx, "upload-1")

			Convey("Then it sees the uploads recorded so far", func() {
				So(err, ShouldBeNil)
				So(object.Namespace, ShouldEqual, "trading-core")
			})
		})

		Convey("When an unknown upload is changed", func() {
			err := store.RecordPart(ctx, "upload-missing", upload.Part{Number: 1}, "2026-01-01T00:00:01Z")

			Convey("Then it reports the upload as not found", func() {
				So(err, ShouldEqual, fileinfostore.ErrUploadNotFound)
			})
		})

		Convey("When an unknown file info is requested", func() {
			_, err := store.GetFileInfo(ctx, "file-missing")

			Convey("Then it reports the file as not found", func() {
				So(err, ShouldEqual, fileinfostore.ErrFileNotFound)
			})
		})
	})

	Convey("Given an event log holding an upload recorded before namespaces existed", t, func() {
		ctx := context.Background()
		log := eventsource.NewInMemoryLog("storage:events")
		_, err := log.Append([]byte(`{"type":"upload_initiated","upload_initiated_event":{"upload_id":"upload-1","user_id":"user-42","key":"user-42/reports/job-1/report.html","content_type":"text/html","created_at":"2026-01-01T00:00:00Z"}}`))
		So(err, ShouldBeNil)
		_, err = log.Append([]byte(`{"type":"part_uploaded","part_uploaded_event":{"upload_id":"upload-1","part_number":1,"size":10,"checksum":"abc","updated_at":"2026-01-01T00:00:01Z"}}`))
		So(err, ShouldBeNil)
		_, err = log.Append([]byte(`{"type":"upload_completed","upload_completed_event":{"upload_id":"upload-1","file_id":"file-1","size":10,"checksum":"abc","updated_at":"2026-01-01T00:00:02Z"}}`))
		So(err, ShouldBeNil)

		Convey("When the store is configured with a legacy namespace", func() {
			store := fileinfostore.NewInMemoryStore(fileinfostore.NewInMemoryStoreInput{
				Log:             log,
				LegacyNamespace: "trading-core",
			})
			fileInfo, err := store.GetFileInfo(ctx, "file-1")

			Convey("Then the legacy file is assigned to that namespace and keeps its stored key", func() {
				So(err, ShouldBeNil)
				So(fileInfo.Namespace, ShouldEqual, "trading-core")
				So(fileInfo.Key, ShouldEqual, "user-42/reports/job-1/report.html")
				So(fileInfo.Size, ShouldEqual, 10)
			})
		})

		Convey("When the store has no legacy namespace", func() {
			store := fileinfostore.NewInMemoryStore(fileinfostore.NewInMemoryStoreInput{Log: log})
			fileInfo, err := store.GetFileInfo(ctx, "file-1")

			Convey("Then the legacy file belongs to no namespace, so no client can reach it", func() {
				So(err, ShouldBeNil)
				So(fileInfo.Namespace, ShouldBeEmpty)
			})
		})
	})
}
