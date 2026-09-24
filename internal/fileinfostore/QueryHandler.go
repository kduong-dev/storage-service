package fileinfostore

import (
	"context"

	"github.com/kduong-dev/storage-service/internal/fileinfo"
	"github.com/kduong-dev/storage-service/internal/upload"
)

// QueryHandler reads upload/file info state from the event log projection.
type QueryHandler interface {
	GetUpload(ctx context.Context, uploadID string) (*upload.Object, error)
	GetFileInfo(ctx context.Context, fileID string) (*fileinfo.FileInfo, error)
}
