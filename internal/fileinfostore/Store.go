package fileinfostore

import (
	"context"

	"github.com/kduong-dev/storage-service/internal/fileinfo"
	"github.com/kduong-dev/storage-service/internal/upload"
)

// Store records the lifecycle of uploads and the file infos they produce.
type Store interface {
	// InitialiseUpload begins a new multipart upload session.
	InitialiseUpload(ctx context.Context, object *upload.Object) error
	// RecordPart records that a part has been successfully stored by the storage.
	RecordPart(ctx context.Context, uploadID string, part upload.Part, updatedAt string) error
	// CompleteUpload finalises the upload, producing a file info.
	CompleteUpload(ctx context.Context, input CompleteUploadInput) error
	// AbortUpload marks the upload as aborted.
	AbortUpload(ctx context.Context, uploadID string, updatedAt string) error
	GetUpload(ctx context.Context, uploadID string) (*upload.Object, error)
	GetFileInfo(ctx context.Context, fileID string) (*fileinfo.FileInfo, error)
}

type CompleteUploadInput struct {
	UploadID  string
	FileID    string
	Size      int64
	Checksum  string
	UpdatedAt string
}
