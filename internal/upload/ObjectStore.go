package upload

import "context"

type ObjectStore interface {
	Initialise(ctx context.Context, object *Object) error
	RecordPart(ctx context.Context, input RecordPartInput) error
	Complete(ctx context.Context, input CompleteInput) error
	Abort(ctx context.Context, uploadID string, updatedAt string) error
	Get(ctx context.Context, uploadID string) (*Object, error)
}

type RecordPartInput struct {
	UploadID  string
	Part      Part
	UpdatedAt string
}

type CompleteInput struct {
	UploadID  string
	FileID    string
	Size      int64
	Checksum  string
	UpdatedAt string
}
