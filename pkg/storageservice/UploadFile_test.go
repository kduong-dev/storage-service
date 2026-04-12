package storageservice_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/kduong/trading-backend/cmd/storage-service/pkg/storageservice"
	. "github.com/smartystreets/goconvey/convey"
)

type fakeClient struct {
	initialisedUploads []fakeInitialiseUploadCall
	uploadedParts      []fakeUploadPartCall
	completedUploads   []string

	initialiseUploadError error
	uploadPartError       error
	completeUploadError   error
}

type fakeInitialiseUploadCall struct {
	filename    string
	contentType string
}

type fakeUploadPartCall struct {
	uploadID   string
	partNumber int
	body       []byte
}

func (client *fakeClient) InitialiseUpload(ctx context.Context, filename string, contentType string) (*storageservice.Upload, error) {
	if client.initialiseUploadError != nil {
		return nil, client.initialiseUploadError
	}
	client.initialisedUploads = append(client.initialisedUploads, fakeInitialiseUploadCall{
		filename:    filename,
		contentType: contentType,
	})
	return &storageservice.Upload{ID: "upload-1"}, nil
}

func (client *fakeClient) UploadPart(ctx context.Context, uploadID string, partNumber int, body io.Reader) (*storageservice.UploadPartResponse, error) {
	if client.uploadPartError != nil {
		return nil, client.uploadPartError
	}
	bodyBytes, err := io.ReadAll(body)
	if err != nil {
		return nil, err
	}
	client.uploadedParts = append(client.uploadedParts, fakeUploadPartCall{
		uploadID:   uploadID,
		partNumber: partNumber,
		body:       bodyBytes,
	})
	return &storageservice.UploadPartResponse{PartNumber: partNumber, Size: int64(len(bodyBytes))}, nil
}

func (client *fakeClient) CompleteUpload(ctx context.Context, uploadID string) (*storageservice.File, error) {
	if client.completeUploadError != nil {
		return nil, client.completeUploadError
	}
	client.completedUploads = append(client.completedUploads, uploadID)
	return &storageservice.File{ID: "file-1", UploadID: uploadID}, nil
}

func (client *fakeClient) DownloadFile(ctx context.Context, fileID string) (*storageservice.DownloadFileResponse, error) {
	return nil, errors.New("not implemented")
}

func TestUploadFile(t *testing.T) {
	Convey("Given a storage service client", t, func() {
		client := &fakeClient{}
		ctx := context.Background()

		Convey("When uploading a file smaller than 5 MB", func() {
			content := strings.Repeat("a", 1024)
			input := storageservice.UploadFileInput{
				Filename:    "small.txt",
				ContentType: "text/plain",
				Body:        strings.NewReader(content),
			}

			file, err := storageservice.UploadFile(ctx, client, input)

			Convey("Then the upload succeeds as a single part", func() {
				So(err, ShouldBeNil)
				So(file, ShouldNotBeNil)
				So(file.ID, ShouldEqual, "file-1")
				So(len(client.uploadedParts), ShouldEqual, 1)
				So(client.uploadedParts[0].partNumber, ShouldEqual, 1)
				So(client.uploadedParts[0].body, ShouldResemble, []byte(content))
				So(len(client.completedUploads), ShouldEqual, 1)
				So(client.completedUploads[0], ShouldEqual, "upload-1")
			})
		})

		Convey("When uploading a file that is exactly 5 MB", func() {
			content := bytes.Repeat([]byte("b"), 5*1024*1024)
			input := storageservice.UploadFileInput{
				Filename:    "exact.bin",
				ContentType: "application/octet-stream",
				Body:        bytes.NewReader(content),
			}

			file, err := storageservice.UploadFile(ctx, client, input)

			Convey("Then the upload succeeds as a single part", func() {
				So(err, ShouldBeNil)
				So(file, ShouldNotBeNil)
				So(len(client.uploadedParts), ShouldEqual, 1)
				So(client.uploadedParts[0].partNumber, ShouldEqual, 1)
				So(len(client.uploadedParts[0].body), ShouldEqual, 5*1024*1024)
			})
		})

		Convey("When uploading a file larger than 5 MB", func() {
			firstPart := bytes.Repeat([]byte("c"), 5*1024*1024)
			secondPart := bytes.Repeat([]byte("d"), 512*1024)
			content := append(firstPart, secondPart...)
			input := storageservice.UploadFileInput{
				Filename:    "large.bin",
				ContentType: "application/octet-stream",
				Body:        bytes.NewReader(content),
			}

			file, err := storageservice.UploadFile(ctx, client, input)

			Convey("Then the upload is split into multiple parts", func() {
				So(err, ShouldBeNil)
				So(file, ShouldNotBeNil)
				So(len(client.uploadedParts), ShouldEqual, 2)
				So(client.uploadedParts[0].partNumber, ShouldEqual, 1)
				So(client.uploadedParts[0].body, ShouldResemble, firstPart)
				So(client.uploadedParts[1].partNumber, ShouldEqual, 2)
				So(client.uploadedParts[1].body, ShouldResemble, secondPart)
				So(len(client.completedUploads), ShouldEqual, 1)
			})
		})

		Convey("When InitialiseUpload returns an error", func() {
			client.initialiseUploadError = errors.New("service unavailable")
			input := storageservice.UploadFileInput{
				Filename:    "file.txt",
				ContentType: "text/plain",
				Body:        strings.NewReader("content"),
			}

			file, err := storageservice.UploadFile(ctx, client, input)

			Convey("Then the error is returned and no parts are uploaded", func() {
				So(err, ShouldNotBeNil)
				So(file, ShouldBeNil)
				So(len(client.uploadedParts), ShouldEqual, 0)
			})
		})

		Convey("When UploadPart returns an error", func() {
			client.uploadPartError = errors.New("write failed")
			input := storageservice.UploadFileInput{
				Filename:    "file.txt",
				ContentType: "text/plain",
				Body:        strings.NewReader("content"),
			}

			file, err := storageservice.UploadFile(ctx, client, input)

			Convey("Then the error is returned and upload is not completed", func() {
				So(err, ShouldNotBeNil)
				So(file, ShouldBeNil)
				So(len(client.completedUploads), ShouldEqual, 0)
			})
		})

		Convey("When CompleteUpload returns an error", func() {
			client.completeUploadError = errors.New("assembly failed")
			input := storageservice.UploadFileInput{
				Filename:    "file.txt",
				ContentType: "text/plain",
				Body:        strings.NewReader("content"),
			}

			file, err := storageservice.UploadFile(ctx, client, input)

			Convey("Then the error is returned", func() {
				So(err, ShouldNotBeNil)
				So(file, ShouldBeNil)
			})
		})
	})
}
