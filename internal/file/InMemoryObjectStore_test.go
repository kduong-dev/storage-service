package file_test

import (
	"context"
	"testing"

	"github.com/kduong-dev/goutil/eventsource"
	"github.com/kduong-dev/storage-service/internal/file"
	. "github.com/smartystreets/goconvey/convey"
)

func TestInMemoryObjectStore(t *testing.T) {
	Convey("Given an in-memory object store backed by an empty event log", t, func() {
		ctx := context.Background()
		log := eventsource.NewInMemoryLog("storage:files")
		store := file.NewInMemoryObjectStore(file.NewInMemoryObjectStoreInput{Log: log})
		object := &file.Object{
			ID:          "file-1",
			Namespace:   "alpha-service",
			UploadID:    "upload-1",
			Key:         "alpha-service/reports/report.html",
			ContentType: "text/html",
			Size:        10,
			Checksum:    "abc",
			CreatedAt:   "2026-01-01T00:00:00Z",
		}
		So(store.Put(ctx, object), ShouldBeNil)

		Convey("When the object is fetched", func() {
			fetched, err := store.Get(ctx, "file-1")

			Convey("Then it matches what was created", func() {
				So(err, ShouldBeNil)
				So(fetched, ShouldResemble, object)
			})
		})

		Convey("When an object with the same ID is put again", func() {
			err := store.Put(ctx, object)

			Convey("Then it is rejected as already existing", func() {
				So(err, ShouldEqual, file.ErrAlreadyExists)
			})
		})

		Convey("When the caller modifies the put or fetched object", func() {
			object.Size = 999
			fetched, err := store.Get(ctx, "file-1")
			So(err, ShouldBeNil)
			fetched.Namespace = "beta-service"

			Convey("Then the stored object is unaffected", func() {
				stored, err := store.Get(ctx, "file-1")
				So(err, ShouldBeNil)
				So(stored.Size, ShouldEqual, 10)
				So(stored.Namespace, ShouldEqual, "alpha-service")
			})
		})

		Convey("When another store is built from the same event log", func() {
			rebuilt := file.NewInMemoryObjectStore(file.NewInMemoryObjectStoreInput{Log: log})
			fetched, err := rebuilt.Get(ctx, "file-1")

			Convey("Then it sees the objects created so far", func() {
				So(err, ShouldBeNil)
				So(fetched.Key, ShouldEqual, "alpha-service/reports/report.html")
			})
		})

		Convey("When an unknown object is requested", func() {
			_, err := store.Get(ctx, "file-missing")

			Convey("Then it reports the file as not found", func() {
				So(err, ShouldEqual, file.ErrNotFound)
			})
		})
	})
}
