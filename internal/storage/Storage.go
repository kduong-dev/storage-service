package storage

import (
	"context"
	"io"
	"os"

	"github.com/kduong-dev/goutil/config"
	"github.com/kduong-dev/goutil/fatal"
)

type Storage interface {
	InitialiseUpload(ctx context.Context, uploadID string) error
	UploadPart(ctx context.Context, input UploadPartInput) (output *UploadPartOutput, err error)
	CompleteUpload(ctx context.Context, input CompleteUploadInput) (output *CompleteUploadOutput, err error)
	AbortUpload(ctx context.Context, uploadID string) error
	OpenFile(fileID string) (io.ReadSeekCloser, error)
	// DeleteFile removes the assembled file; a missing file is not an error.
	DeleteFile(ctx context.Context, fileID string) error
}

type UploadPartInput struct {
	UploadID   string
	PartNumber int
	Reader     io.Reader
}

type UploadPartOutput struct {
	Size     int64
	Checksum string
}

type CompleteUploadInput struct {
	UploadID    string
	FileID      string
	PartNumbers []int
}

type CompleteUploadOutput struct {
	Size     int64
	Checksum string
}

func FromEnv() Storage {
	storage := config.EnvString("STORAGE", "FILESYSTEM")
	switch storage {
	case "FILESYSTEM":
		root := config.EnvString("STORAGE_FILESYSTEM_DIRECTORY", "./tmp/storage")
		fatal.OnError(os.MkdirAll(root, 0o755))
		return NewFileSystemStorage(NewFileSystemStorageInput{
			Root: root,
		})
	default:
		fatal.LogErrorf("unsupported storage type: %s", storage)
		return nil
	}
}
