package upload_test

import (
	"context"
	"sync"
	"testing"

	"github.com/kduong-dev/goutil/eventsource"
	"github.com/kduong-dev/storage-service/internal/upload"
	"github.com/kduong-dev/storage-service/pkg/storageservice"
	. "github.com/smartystreets/goconvey/convey"
)

func TestObjectStoreThreadSafeDecorator(t *testing.T) {
	Convey("Given an initialised upload in a store wrapped in the thread safe decorator", t, func() {
		ctx := context.Background()
		store := upload.NewObjectStoreThreadSafeDecorator(upload.NewObjectStoreThreadSafeDecoratorInput{
			Decorated: upload.NewEventSourcedObjectStore(upload.NewEventSourcedObjectStoreInput{
				Log: eventsource.NewInMemoryLog("storage:uploads"),
			}),
		})
		So(store.Initialise(ctx, &storageservice.UploadObject{ID: "upload-1"}), ShouldBeNil)
		Convey("When parts are recorded concurrently", func() {
			var waitGroup sync.WaitGroup
			errs := make([]error, 50)
			for index := range errs {
				waitGroup.Go(func() {
					errs[index] = store.RecordPart(ctx, upload.RecordPartInput{
						UploadID:  "upload-1",
						Part:      storageservice.Part{PartNumber: index + 1},
						UpdatedAt: "2026-01-01T00:00:01Z",
					})
				})
			}
			waitGroup.Wait()
			Convey("Then every part is recorded", func() {
				for _, err := range errs {
					So(err, ShouldBeNil)
				}
				object, err := store.Get(ctx, "upload-1")
				So(err, ShouldBeNil)
				So(len(object.Parts), ShouldEqual, 50)
			})
		})
	})
}
