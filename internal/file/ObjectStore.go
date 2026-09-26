package file

import (
	"context"

	"github.com/kduong-dev/storage-service/pkg/storageservice"
)

type ObjectStore interface {
	Put(ctx context.Context, object *storageservice.FileObject) error
	Get(ctx context.Context, fileID string) (*storageservice.FileObject, error)
	List(ctx context.Context, input ListInput) (*storageservice.ListFileObjectsOutput, error)
	Delete(ctx context.Context, fileID string) error
}

// ListInput pages through objects ordered by key, then ID. After is the ID of
// the last object on the previous page, empty for the first page.
type ListInput struct {
	KeyPrefix string
	After     string
	Limit     int
}
