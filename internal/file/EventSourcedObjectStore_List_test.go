package file_test

import (
	"context"
	"testing"

	"github.com/kduong-dev/goutil/eventsource"
	"github.com/kduong-dev/storage-service/internal/file"
	"github.com/kduong-dev/storage-service/pkg/storageservice"
	. "github.com/smartystreets/goconvey/convey"
)

func TestEventSourcedObjectStoreList(t *testing.T) {
	Convey("Given an event-sourced object store holding files across two namespaces", t, func() {
		ctx := context.Background()
		store := file.NewEventSourcedObjectStore(file.NewEventSourcedObjectStoreInput{
			Log: eventsource.NewInMemoryLog("storage:files"),
		})
		for _, object := range []*storageservice.File{
			{ID: "file-b", Key: "alpha-service/reports/b.html"},
			{ID: "file-a", Key: "alpha-service/reports/a.html"},
			{ID: "file-d", Key: "alpha-service/images/logo.png"},
			{ID: "file-c", Key: "alpha-service/reports/b.html"},
			{ID: "file-e", Key: "beta-service/reports/a.html"},
		} {
			So(store.Put(ctx, object), ShouldBeNil)
		}
		listIDs := func(input file.ListInput) ([]string, string) {
			output, err := store.List(ctx, input)
			So(err, ShouldBeNil)
			ids := make([]string, len(output.Objects))
			for index, object := range output.Objects {
				ids[index] = object.ID
			}
			return ids, output.NextAfter
		}
		Convey("When listing a namespace in one page", func() {
			ids, nextAfter := listIDs(file.ListInput{KeyPrefix: "alpha-service/", Limit: 10})
			Convey("Then only that namespace is returned, ordered by key then ID, with no next page", func() {
				So(ids, ShouldResemble, []string{"file-d", "file-a", "file-b", "file-c"})
				So(nextAfter, ShouldBeEmpty)
			})
		})
		Convey("When listing a narrower prefix", func() {
			ids, _ := listIDs(file.ListInput{KeyPrefix: "alpha-service/reports/", Limit: 10})
			Convey("Then only keys under it are returned", func() {
				So(ids, ShouldResemble, []string{"file-a", "file-b", "file-c"})
			})
		})
		Convey("When paging through a namespace two at a time", func() {
			firstIDs, firstNext := listIDs(file.ListInput{KeyPrefix: "alpha-service/", Limit: 2})
			secondIDs, secondNext := listIDs(file.ListInput{KeyPrefix: "alpha-service/", After: firstNext, Limit: 2})
			Convey("Then the pages cover every file once, splitting files that share a key", func() {
				So(firstIDs, ShouldResemble, []string{"file-d", "file-a"})
				So(firstNext, ShouldEqual, "file-a")
				So(secondIDs, ShouldResemble, []string{"file-b", "file-c"})
				So(secondNext, ShouldBeEmpty)
			})
		})
		Convey("When a new file sorting before the cursor is added between pages", func() {
			_, firstNext := listIDs(file.ListInput{KeyPrefix: "alpha-service/", Limit: 2})
			So(store.Put(ctx, &storageservice.File{ID: "file-f", Key: "alpha-service/aaa.html"}), ShouldBeNil)
			secondIDs, _ := listIDs(file.ListInput{KeyPrefix: "alpha-service/", After: firstNext, Limit: 10})
			Convey("Then the next page continues after the cursor without repeating files", func() {
				So(secondIDs, ShouldResemble, []string{"file-b", "file-c"})
			})
		})
		Convey("When the after cursor belongs to another namespace", func() {
			_, err := store.List(ctx, file.ListInput{KeyPrefix: "alpha-service/", After: "file-e", Limit: 10})
			Convey("Then it is rejected as invalid", func() {
				So(err, ShouldEqual, file.ErrInvalidAfter)
			})
		})
		Convey("When the after cursor does not exist", func() {
			_, err := store.List(ctx, file.ListInput{KeyPrefix: "alpha-service/", After: "file-missing", Limit: 10})
			Convey("Then it is rejected as invalid", func() {
				So(err, ShouldEqual, file.ErrInvalidAfter)
			})
		})
		Convey("When a listed object is modified by the caller", func() {
			output, err := store.List(ctx, file.ListInput{KeyPrefix: "alpha-service/", Limit: 1})
			So(err, ShouldBeNil)
			output.Objects[0].Key = "beta-service/stolen.png"
			Convey("Then the stored object is unaffected", func() {
				stored, err := store.Get(ctx, "file-d")
				So(err, ShouldBeNil)
				So(stored.Key, ShouldEqual, "alpha-service/images/logo.png")
			})
		})
	})
}
