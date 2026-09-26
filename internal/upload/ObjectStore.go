package upload

import (
	"context"

	"github.com/kduong-dev/storage-service/pkg/storageservice"
)

type ObjectStore interface {
	Initialise(ctx context.Context, object *storageservice.UploadObject) error
	RecordPart(ctx context.Context, input RecordPartInput) error
	Complete(ctx context.Context, input CompleteInput) error
	Abort(ctx context.Context, input AbortInput) error
	Get(ctx context.Context, uploadID string) (*storageservice.UploadObject, error)
}

type RecordPartInput struct {
	UploadID  string
	Part      storageservice.Part
	UpdatedAt string
}

type CompleteInput struct {
	UploadID  string
	FileID    string
	Size      int64
	Checksum  string
	UpdatedAt string
}

type AbortInput struct {
	UploadID  string
	UpdatedAt string
}
