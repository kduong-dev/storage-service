package fileinfostore

import (
	"context"

	"github.com/kduong-dev/storage-service/internal/upload"
)

// CommandHandler mutates upload/file info state via the event log.
type CommandHandler interface {
	// InitialiseUpload begins a new multipart upload session.
	InitialiseUpload(ctx context.Context, object *upload.Object) error
	// RecordPart records that a part has been successfully stored by the storage.
	RecordPart(ctx context.Context, uploadID string, part upload.Part, updatedAt string) error
	// CompleteUpload finalises the upload, producing a stored file info record.
	CompleteUpload(ctx context.Context, input CompleteUploadInput) error
	// AbortUpload marks the upload as aborted.
	AbortUpload(ctx context.Context, uploadID string, updatedAt string) error
}

type CompleteUploadInput struct {
	UploadID  string
	FileID    string
	Size      int64
	Checksum  string
	UpdatedAt string
}
