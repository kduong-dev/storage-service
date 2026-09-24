package file

import "context"

type ObjectStore interface {
	Put(ctx context.Context, object *Object) error
	Get(ctx context.Context, fileID string) (*Object, error)
	List(ctx context.Context, input ListInput) (*ListOutput, error)
}

// ListInput pages through objects ordered by key, then ID. After is the ID of
// the last object on the previous page, empty for the first page.
type ListInput struct {
	KeyPrefix string
	After     string
	Limit     int
}

type ListOutput struct {
	Objects []*Object
	// NextAfter is the After for the next page, empty when there are no more.
	NextAfter string
}
