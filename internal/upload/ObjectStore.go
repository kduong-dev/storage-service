package upload

import "context"

type ObjectStore interface {
	Initialise(ctx context.Context, object *Object) error
	RecordPart(ctx context.Context, uploadID string, part Part, updatedAt string) error
	Complete(ctx context.Context, uploadID string, updatedAt string) error
	Abort(ctx context.Context, uploadID string, updatedAt string) error
	Get(ctx context.Context, uploadID string) (*Object, error)
}
