package storage

import (
	"io"
	"os"

	"github.com/kduong/trading-backend/internal/config"
	"github.com/kduong/trading-backend/internal/fatal"
)

// Backend stores and retrieves the raw bytes of uploaded files.
// It is intentionally decoupled from the event-sourced filestore; the filestore
// tracks metadata and integrity, while the backend handles the blob data.
type Backend interface {
	// WritePart persists a single part for the given upload.
	// partNumber is 1-based, matching the S3 multipart convention.
	WritePart(uploadID string, partNumber int, r io.Reader) (size int64, checksum string, err error)

	// Assemble concatenates all parts (in ascending part-number order) into a
	// single object identified by fileID and returns the total size and overall
	// MD5 checksum. The assembled object must then be retrievable via Open.
	Assemble(uploadID string, fileID string, partNumbers []int) (size int64, checksum string, err error)

	// Open returns a ReadSeekCloser for the assembled object so that HTTP range
	// requests are supported out-of-the-box.
	Open(fileID string) (io.ReadSeekCloser, error)

	// DeleteParts removes the temporary part data for an upload (called on abort
	// or after a successful assembly).
	DeleteParts(uploadID string) error
}

func FromEnv() Backend {
	backendType := config.EnvString("STORAGE_BACKEND", "INMEMORY")
	switch backendType {
	case "FILESYSTEM":
		storageDir := config.EnvString("STORAGE_FILESYSTEM_DIRECTORY", "./tmp/storage")
		fatal.OnError(os.MkdirAll(storageDir, 0o755))
		return NewFileSystemBackend(storageDir)
	case "INMEMORY":
		return NewInMemoryBackend()
	default:
		panic("unsupported storage backend type: " + backendType)
	}
}
