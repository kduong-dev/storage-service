package storage_test

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/iotest"

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
						PartNumbers: []int{2, 1},
					})
					So(err, ShouldBeNil)
					So(output.Size, ShouldEqual, len("hello world"))
					readSeekCloser, err := fileSystemStorage.OpenFile("file-1")
					So(err, ShouldBeNil)
					defer readSeekCloser.Close()
					content, err := io.ReadAll(readSeekCloser)
					So(err, ShouldBeNil)
					So(string(content), ShouldEqual, "hello world")
					Convey("And the file is deleted", func() {
						So(fileSystemStorage.DeleteFile(ctx, "file-1"), ShouldBeNil)
						Convey("Then it can no longer be opened", func() {
							_, err := fileSystemStorage.OpenFile("file-1")
							So(errors.Is(err, os.ErrNotExist), ShouldBeTrue)
						})
						Convey("Then deleting it again succeeds", func() {
							So(fileSystemStorage.DeleteFile(ctx, "file-1"), ShouldBeNil)
						})
					})
				})
			})
			Convey("And the upload is aborted", func() {
				So(fileSystemStorage.AbortUpload(ctx, "upload-1"), ShouldBeNil)
				Convey("Then uploading a part reports the upload as not found", func() {
					_, err := fileSystemStorage.UploadPart(ctx, storage.UploadPartInput{
						UploadID:   "upload-1",
						PartNumber: 1,
						Reader:     strings.NewReader("late"),
					})
					So(err, ShouldEqual, storage.ErrUploadNotFound)
				})
				Convey("Then completing it reports the upload as not found", func() {
					_, err := fileSystemStorage.CompleteUpload(ctx, storage.CompleteUploadInput{
						UploadID:    "upload-1",
						FileID:      "file-1",
						PartNumbers: []int{1},
					})
					So(err, ShouldEqual, storage.ErrUploadNotFound)
				})
			})
			Convey("And a part body fails while being read", func() {
				_, err := fileSystemStorage.UploadPart(ctx, storage.UploadPartInput{
					UploadID:   "upload-1",
					PartNumber: 1,
					Reader:     io.MultiReader(strings.NewReader("partial"), iotest.ErrReader(errors.New("connection reset"))),
				})
				Convey("Then the read error is returned instead of stopping the server", func() {
					So(err, ShouldNotBeNil)
				})
			})
		})
	})
}
