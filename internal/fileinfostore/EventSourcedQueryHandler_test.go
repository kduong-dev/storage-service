package fileinfostore_test

import (
	"context"
	"testing"

	"github.com/kduong-dev/goutil/eventsource"
	"github.com/kduong-dev/storage-service/internal/fileinfostore"
	. "github.com/smartystreets/goconvey/convey"
)

func TestEventSourcedQueryHandler(t *testing.T) {
	Convey("Given an event log holding an upload recorded before namespaces existed", t, func() {
		ctx := context.Background()
		log := eventsource.NewInMemoryLog("storage:events")
		_, err := log.Append([]byte(`{"type":"upload_initiated","upload_initiated_event":{"upload_id":"upload-1","user_id":"user-42","key":"user-42/reports/job-1/report.html","content_type":"text/html","created_at":"2026-01-01T00:00:00Z"}}`))
		So(err, ShouldBeNil)
		_, err = log.Append([]byte(`{"type":"part_uploaded","part_uploaded_event":{"upload_id":"upload-1","part_number":1,"size":10,"checksum":"abc","updated_at":"2026-01-01T00:00:01Z"}}`))
		So(err, ShouldBeNil)
		_, err = log.Append([]byte(`{"type":"upload_completed","upload_completed_event":{"upload_id":"upload-1","file_id":"file-1","size":10,"checksum":"abc","updated_at":"2026-01-01T00:00:02Z"}}`))
		So(err, ShouldBeNil)

		Convey("When the query handler is configured with a legacy namespace", func() {
			queryHandler := fileinfostore.NewEventSourcedQueryHandler(fileinfostore.NewEventSourcedQueryHandlerInput{
				Log:             log,
				LegacyNamespace: "trading-core",
			})
			fileInfo, err := queryHandler.GetFileInfo(ctx, "file-1")

			Convey("Then the legacy file is assigned to that namespace and keeps its stored key", func() {
				So(err, ShouldBeNil)
				So(fileInfo.Namespace, ShouldEqual, "trading-core")
				So(fileInfo.Key, ShouldEqual, "user-42/reports/job-1/report.html")
				So(fileInfo.Size, ShouldEqual, 10)
			})
		})

		Convey("When the query handler has no legacy namespace", func() {
			queryHandler := fileinfostore.NewEventSourcedQueryHandler(fileinfostore.NewEventSourcedQueryHandlerInput{Log: log})
			fileInfo, err := queryHandler.GetFileInfo(ctx, "file-1")

			Convey("Then the legacy file belongs to no namespace, so no client can reach it", func() {
				So(err, ShouldBeNil)
				So(fileInfo.Namespace, ShouldBeEmpty)
			})
		})

		Convey("When an unknown file is requested", func() {
			queryHandler := fileinfostore.NewEventSourcedQueryHandler(fileinfostore.NewEventSourcedQueryHandlerInput{Log: log})
			_, err := queryHandler.GetFileInfo(ctx, "file-missing")

			Convey("Then it reports the file as not found", func() {
				So(err, ShouldEqual, fileinfostore.ErrFileNotFound)
			})
		})
	})
}
