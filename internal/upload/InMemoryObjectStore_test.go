package upload_test

import (
	"context"
	"testing"

	"github.com/kduong-dev/goutil/eventsource"
	"github.com/kduong-dev/storage-service/internal/upload"
	. "github.com/smartystreets/goconvey/convey"
)

func TestInMemoryObjectStore(t *testing.T) {
	Convey("Given an in-memory object store with an initialised upload", t, func() {
		ctx := context.Background()
		log := eventsource.NewInMemoryLog("storage:uploads")
		store := upload.NewInMemoryObjectStore(upload.NewInMemoryObjectStoreInput{Log: log})
		object := &upload.Object{
			ID:          "upload-1",
			Namespace:   "alpha-service",
			Key:         "alpha-service/reports/report.html",
			ContentType: "text/html",
			CreatedAt:   "2026-01-01T00:00:00Z",
		}
		So(store.Initialise(ctx, object), ShouldBeNil)

		Convey("When it is initialised again", func() {
			err := store.Initialise(ctx, object)

			Convey("Then it is rejected as already existing", func() {
				So(err, ShouldEqual, upload.ErrAlreadyExists)
			})
		})

		Convey("When a part is recorded twice under the same number", func() {
			So(store.RecordPart(ctx, "upload-1", upload.Part{Number: 1, Size: 5, Checksum: "first"}, "2026-01-01T00:00:01Z"), ShouldBeNil)
			So(store.RecordPart(ctx, "upload-1", upload.Part{Number: 1, Size: 7, Checksum: "second"}, "2026-01-01T00:00:02Z"), ShouldBeNil)
			fetched, err := store.Get(ctx, "upload-1")

			Convey("Then the later part replaces the earlier one", func() {
				So(err, ShouldBeNil)
				So(fetched.Parts, ShouldResemble, []upload.Part{{Number: 1, Size: 7, Checksum: "second"}})
				So(fetched.UpdatedAt, ShouldEqual, "2026-01-01T00:00:02Z")
			})
		})

		Convey("When the upload is completed", func() {
			So(store.Complete(ctx, "upload-1", "2026-01-01T00:00:03Z"), ShouldBeNil)

			Convey("Then it is completed and accepts no more changes", func() {
				fetched, err := store.Get(ctx, "upload-1")
				So(err, ShouldBeNil)
				So(fetched.Status, ShouldEqual, upload.StatusCompleted)
				So(store.RecordPart(ctx, "upload-1", upload.Part{Number: 2}, "2026-01-01T00:00:04Z"), ShouldEqual, upload.ErrNotActive)
				So(store.Abort(ctx, "upload-1", "2026-01-01T00:00:04Z"), ShouldEqual, upload.ErrNotActive)
			})
		})

		Convey("When the upload is aborted", func() {
			So(store.Abort(ctx, "upload-1", "2026-01-01T00:00:03Z"), ShouldBeNil)

			Convey("Then it is aborted and cannot be completed", func() {
				fetched, err := store.Get(ctx, "upload-1")
				So(err, ShouldBeNil)
				So(fetched.Status, ShouldEqual, upload.StatusAborted)
				So(store.Complete(ctx, "upload-1", "2026-01-01T00:00:04Z"), ShouldEqual, upload.ErrNotActive)
			})
		})

		Convey("When a fetched upload is modified by the caller", func() {
			So(store.RecordPart(ctx, "upload-1", upload.Part{Number: 1, Size: 5}, "2026-01-01T00:00:01Z"), ShouldBeNil)
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
			rebuilt := upload.NewInMemoryObjectStore(upload.NewInMemoryObjectStoreInput{Log: log})
			fetched, err := rebuilt.Get(ctx, "upload-1")

			Convey("Then it sees the uploads recorded so far", func() {
				So(err, ShouldBeNil)
				So(fetched.Namespace, ShouldEqual, "alpha-service")
			})
		})

		Convey("When an unknown upload is changed", func() {
			err := store.RecordPart(ctx, "upload-missing", upload.Part{Number: 1}, "2026-01-01T00:00:01Z")

			Convey("Then it reports the upload as not found", func() {
				So(err, ShouldEqual, upload.ErrNotFound)
			})
		})
	})
}
