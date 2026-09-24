package file

import "context"

type ObjectStore interface {
	Create(ctx context.Context, object *Object) error
	Get(ctx context.Context, fileID string) (*Object, error)
}
