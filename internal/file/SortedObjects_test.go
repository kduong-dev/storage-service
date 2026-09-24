package file_test

import (
	"testing"

	"github.com/kduong-dev/storage-service/internal/file"
	"github.com/kduong-dev/storage-service/pkg/storageservice"
	. "github.com/smartystreets/goconvey/convey"
)

func TestSortedObjects(t *testing.T) {
	Convey("Given sorted objects built from inserts in arbitrary order", t, func() {
		var objects file.SortedObjects
		for _, object := range []*storageservice.File{
			{ID: "file-c", Key: "alpha/reports/b.html"},
			{ID: "file-a", Key: "alpha/reports/a.html"},
			{ID: "file-e", Key: "beta/reports/a.html"},
			{ID: "file-d", Key: "alpha/images/logo.png"},
			{ID: "file-b", Key: "alpha/reports/b.html"},
		} {
			objects.Add(object)
		}
		ids := func(objects []*storageservice.File) []string {
			result := make([]string, len(objects))
			for index, object := range objects {
				result[index] = object.ID
			}
			return result
		}
		Convey("When reading every object in one page", func() {
			page, _ := objects.Page(file.PageInput{Limit: 10})
			Convey("Then they are ordered by key, then by ID for shared keys", func() {
				So(ids(page), ShouldResemble, []string{"file-d", "file-a", "file-b", "file-c", "file-e"})
			})
		})
		Convey("When an object is added after a read has sorted them", func() {
			objects.Page(file.PageInput{Limit: 10})
			objects.Add(&storageservice.File{ID: "file-f", Key: "alpha/aaa.html"})
			page, _ := objects.Page(file.PageInput{Limit: 10})
			Convey("Then the next read places it in order", func() {
				So(ids(page), ShouldResemble, []string{"file-f", "file-d", "file-a", "file-b", "file-c", "file-e"})
			})
		})
		Convey("When an object sharing its key with another is removed", func() {
			objects.Remove(&storageservice.File{ID: "file-b", Key: "alpha/reports/b.html"})
			page, _ := objects.Page(file.PageInput{Limit: 10})
			Convey("Then only that object is gone and the rest stay in order", func() {
				So(ids(page), ShouldResemble, []string{"file-d", "file-a", "file-c", "file-e"})
			})
		})
		Convey("When an absent object is removed", func() {
			objects.Remove(&storageservice.File{ID: "file-z", Key: "zeta/a.html"})
			page, _ := objects.Page(file.PageInput{Limit: 10})
			Convey("Then nothing changes", func() {
				So(ids(page), ShouldResemble, []string{"file-d", "file-a", "file-b", "file-c", "file-e"})
			})
		})
		Convey("When finding where a key prefix starts", func() {
			Convey("Then it is the first object whose key is not before the prefix", func() {
				So(objects.IndexOfKeyPrefix("alpha/reports/"), ShouldEqual, 1)
				So(objects.IndexOfKeyPrefix("beta/"), ShouldEqual, 4)
				So(objects.IndexOfKeyPrefix("gamma/"), ShouldEqual, 5)
			})
		})
		Convey("When finding the index after an object", func() {
			Convey("Then it is just past the object, or where it would be inserted if absent", func() {
				So(objects.IndexAfter(&storageservice.File{ID: "file-b", Key: "alpha/reports/b.html"}), ShouldEqual, 3)
				So(objects.IndexAfter(&storageservice.File{ID: "file-bb", Key: "alpha/reports/b.html"}), ShouldEqual, 3)
				So(objects.IndexAfter(&storageservice.File{ID: "file-z", Key: "zeta/a.html"}), ShouldEqual, 5)
			})
		})
		Convey("When taking a page under a prefix", func() {
			page, hasMore := objects.Page(file.PageInput{KeyPrefix: "alpha/", Limit: 2})
			Convey("Then it stops at the limit and reports that more follow", func() {
				So(ids(page), ShouldResemble, []string{"file-d", "file-a"})
				So(hasMore, ShouldBeTrue)
			})
		})
		Convey("When taking the page after the last object on a previous page", func() {
			page, hasMore := objects.Page(file.PageInput{KeyPrefix: "alpha/", After: &storageservice.File{ID: "file-a", Key: "alpha/reports/a.html"}, Limit: 2})
			Convey("Then it continues within the prefix and reports no more once the prefix ends", func() {
				So(ids(page), ShouldResemble, []string{"file-b", "file-c"})
				So(hasMore, ShouldBeFalse)
			})
		})
		Convey("When taking a page under a prefix with no objects", func() {
			page, hasMore := objects.Page(file.PageInput{KeyPrefix: "gamma/", Limit: 2})
			Convey("Then it is empty with no more to follow", func() {
				So(page, ShouldBeEmpty)
				So(hasMore, ShouldBeFalse)
			})
		})
	})
}
