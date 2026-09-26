package file

import (
	"context"

	"github.com/kduong-dev/storage-service/pkg/storageservice"
)

type ObjectStore interface {
	Put(ctx context.Context, object *storageservice.FileObject) error
	Get(ctx context.Context, fileID string) (*storageservice.FileObject, error)
	List(ctx context.Context, input ListInput) (*storageservice.ListFileObjectsOutput, error)
	// Move changes the file's key, which renames it or moves it to another
	// directory; the stored bytes are keyed by file ID and stay where they are.
	Move(ctx context.Context, input MoveInput) error
	Delete(ctx context.Context, fileID string) error
}

type MoveInput struct {
	FileID string
	Key    string
}

// ListInput pages through objects ordered by key, then ID. After is the ID of
// the last object on the previous page, empty for the first page.
type ListInput struct {
	KeyPrefix string
	After     string
	Limit     int
}
