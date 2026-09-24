package file

import "context"

type ObjectStore interface {
	Put(ctx context.Context, object *Object) error
	Get(ctx context.Context, fileID string) (*Object, error)
}
