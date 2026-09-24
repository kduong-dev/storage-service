package storage_test

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kduong-dev/storage-service/internal/storage"
	. "github.com/smartystreets/goconvey/convey"
)

func TestFileSystemStorage(t *testing.T) {
	Convey("Given a file system storage", t, func() {
		ctx := context.Background()
		root := t.TempDir()
		fileSystemStorage := storage.NewFileSystemStorage(storage.NewFileSystemStorageInput{Root: root})
		Convey("When an upload is initialised", func() {
			err := fileSystemStorage.InitialiseUpload(ctx, "upload-1")
			So(err, ShouldBeNil)
			Convey("Then its parts directory exists", func() {
				info, err := os.Stat(filepath.Join(root, "parts", "upload-1"))
				So(err, ShouldBeNil)
				So(info.IsDir(), ShouldBeTrue)
			})
			Convey("And parts are written out of order", func() {
				for partNumber, body := range map[int]string{2: "world", 1: "hello "} {
					output, err := fileSystemStorage.UploadPart(ctx, storage.UploadPartInput{
						UploadID:   "upload-1",
						PartNumber: partNumber,
						Reader:     strings.NewReader(body),
					})
					So(err, ShouldBeNil)
					So(output.Size, ShouldEqual, len(body))
				}
				Convey("Then assembling them produces the file in part order", func() {
					output, err := fileSystemStorage.CompleteUpload(ctx, storage.CompleteUploadInput{
						UploadID:    "upload-1",
						FileID:      "file-1",
						Key:         "alpha-service/reports/report.txt",
						PartNumbers: []int{2, 1},
					})
					So(err, ShouldBeNil)
					So(output.Size, ShouldEqual, len("hello world"))
					readSeekCloser, err := fileSystemStorage.OpenFile("alpha-service/reports/report.txt")
					So(err, ShouldBeNil)
					defer readSeekCloser.Close()
					content, err := io.ReadAll(readSeekCloser)
					So(err, ShouldBeNil)
					So(string(content), ShouldEqual, "hello world")
				})
			})
		})
	})
}
