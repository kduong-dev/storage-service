package storageservice

import (
	"context"
	"io"
	"time"

	"github.com/kduong-dev/goutil/config"
	"github.com/kduong-dev/goutil/fatal"
)

// UploadObject tracks a multipart upload session.
type UploadObject struct {
	ID          string `json:"id"`
	Key         string `json:"key"`
	ContentType string `json:"content_type"`
	// Parts received so far, indexed by part number (1-based).
	Parts     []Part `json:"parts,omitempty"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

// Part describes one uploaded chunk.
type Part struct {
	PartNumber int    `json:"part_number"`
	Size       int64  `json:"size"`
	Checksum   string `json:"checksum"` // hex-encoded MD5 of the part bytes
}

// FileObject is the completed, stored object produced after an upload is finalised.
type FileObject struct {
	ID          string `json:"id"`
	Key         string `json:"key"`
	ContentType string `json:"content_type"`
	Size        int64  `json:"size"`
	Checksum    string `json:"checksum"` // hex-encoded MD5 of the full file
	CreatedAt   string `json:"created_at"`
}

// Client is the public interface for the storage-service API. Every call is
// scoped to the namespace of the API key the client was configured with;
// deciding which end user may access a file is the calling service's job.
type Client interface {
	// InitialiseUpload begins a new multipart upload session. key is a
	// relative path within the caller's namespace.
	InitialiseUpload(ctx context.Context, input InitialiseUploadInput) (*UploadObject, error)

	// UploadPart streams one chunk to an existing upload session.
	UploadPart(ctx context.Context, input UploadPartInput) (*UploadPartOutput, error)

	// CompleteUpload finalises an upload session and assembles all parts into a FileObject.
	CompleteUpload(ctx context.Context, input CompleteUploadInput) (*FileObject, error)

	// AbortUpload cancels an upload session and discards its uploaded parts.
	AbortUpload(ctx context.Context, input AbortUploadInput) error

	// DownloadFile streams the assembled file for the given file ID.
	// The caller is responsible for closing DownloadFileOutput.Body.
	DownloadFile(ctx context.Context, input DownloadFileInput) (*DownloadFileOutput, error)

	// ListFileObjects returns one page of the caller's files, ordered by key. Pass
	// the returned NextCursor back as Cursor to fetch the next page.
	ListFileObjects(ctx context.Context, input ListFileObjectsInput) (*ListFileObjectsOutput, error)

	// DeleteFile removes the file and its metadata.
	DeleteFile(ctx context.Context, input DeleteFileInput) error
}

// InitialiseUploadInput is also the request body sent to the server.
type InitialiseUploadInput struct {
	Key         string `json:"key"`
	ContentType string `json:"content_type"`
}

type UploadPartInput struct {
	UploadID   string
	PartNumber int
	Body       io.Reader
}

type UploadPartOutput struct {
	PartNumber int    `json:"part_number"`
	Size       int64  `json:"size"`
	Checksum   string `json:"checksum"`
}

type CompleteUploadInput struct {
	UploadID string
}

type AbortUploadInput struct {
	UploadID string
}

type DownloadFileInput struct {
	FileID string
}

type DownloadFileOutput struct {
	ContentType        string
	ContentDisposition string
	Body               io.ReadCloser
}

type DeleteFileInput struct {
	FileID string
}

const (
	DefaultListFileObjectsLimit = 100
	MaxListFileObjectsLimit     = 1000
)

type ListFileObjectsInput struct {
	// Prefix filters to keys starting with it, relative to the caller's namespace.
	Prefix string
	Cursor string
	// Limit is the page size; zero uses DefaultListFileObjectsLimit, at most
	// MaxListFileObjectsLimit.
	Limit int
}

type ListFileObjectsOutput struct {
	Files []*FileObject `json:"files"`
	// NextCursor is empty on the last page.
	NextCursor string `json:"next_cursor,omitempty"`
}

func ClientFromEnv() Client {
	implementation := config.EnvStringOrFatal("STORAGE_SERVICE_CLIENT_IMPLEMENTATION")
	switch implementation {
	case "HTTP":
		return NewHTTPClient(NewHTTPClientInput{
			Timeout: config.EnvDuration("STORAGE_SERVICE_HTTP_CLIENT_TIMEOUT", 20*time.Second),
			BaseURL: config.EnvURLOrFatal("STORAGE_SERVICE_URL"),
			APIKey:  config.EnvStringOrFatal("STORAGE_SERVICE_API_KEY"),
		})
	default:
		fatal.LogErrorf("invalid storage service client implementation: %s", implementation)
		return nil
	}
}
