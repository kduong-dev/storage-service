package file_test

import (
	"context"
	"strconv"
	"sync"
	"testing"

	"github.com/kduong-dev/goutil/eventsource"
	"github.com/kduong-dev/storage-service/internal/file"
	. "github.com/smartystreets/goconvey/convey"
)

func TestObjectStoreThreadSafeDecorator(t *testing.T) {
	Convey("Given an event-sourced object store wrapped in the thread safe decorator", t, func() {
		ctx := context.Background()
		store := file.NewObjectStoreThreadSafeDecorator(file.NewObjectStoreThreadSafeDecoratorInput{
			Decorated: file.NewEventSourcedObjectStore(file.NewEventSourcedObjectStoreInput{
				Log: eventsource.NewInMemoryLog("storage:files"),
			}),
		})

		Convey("When objects are put and fetched concurrently", func() {
			var waitGroup sync.WaitGroup
			errs := make([]error, 50)
			for index := range errs {
				waitGroup.Go(func() {
					fileID := "file-" + strconv.Itoa(index)
					if errs[index] = store.Put(ctx, &file.Object{ID: fileID}); errs[index] == nil {
						_, errs[index] = store.Get(ctx, fileID)
					}
				})
			}
			waitGroup.Wait()

			Convey("Then every object is stored and readable", func() {
				for _, err := range errs {
					So(err, ShouldBeNil)
				}
			})
		})
	})
}
