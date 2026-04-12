package filestore

import "context"

// CommandHandler mutates upload/file state via the event log.
type CommandHandler interface {
	// InitialiseUpload begins a new multipart upload session.
	InitialiseUpload(ctx context.Context, upload *Upload) error
	// RecordPart records that a part has been successfully stored by the backend.
	RecordPart(ctx context.Context, uploadID string, part Part, updatedAt string) error
	// CompleteUpload finalises the upload, producing a stored File record.
	CompleteUpload(ctx context.Context, uploadID string, fileID string, size int64, checksum string, updatedAt string) error
	// AbortUpload marks the upload as aborted.
	AbortUpload(ctx context.Context, uploadID string, updatedAt string) error
}
