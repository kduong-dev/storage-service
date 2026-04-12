package filestore

import "context"

// QueryHandler reads upload/file state from the event log projection.
type QueryHandler interface {
	GetUpload(ctx context.Context, uploadID string) (*Upload, error)
	GetFile(ctx context.Context, fileID string) (*File, error)
}
