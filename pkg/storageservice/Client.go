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

	// GetUploadObject returns an in-progress upload session, including the parts
	// received so far, so an interrupted upload can be resumed.
	GetUploadObject(ctx context.Context, input GetUploadObjectInput) (*UploadObject, error)

	// AbortUpload cancels an upload session and discards its uploaded parts.
	AbortUpload(ctx context.Context, input AbortUploadInput) error

	// DownloadFile streams the assembled file for the given file ID, or the
	// requested byte range of it. The caller is responsible for closing
	// DownloadFileOutput.Body.
	DownloadFile(ctx context.Context, input DownloadFileInput) (*DownloadFileOutput, error)

	// GetFileObject returns the file's metadata without downloading it.
	GetFileObject(ctx context.Context, input GetFileObjectInput) (*FileObject, error)

	// ListFileObjects returns one page of the caller's files, ordered by key. Pass
	// the returned NextCursor back as Cursor to fetch the next page.
	ListFileObjects(ctx context.Context, input ListFileObjectsInput) (*ListFileObjectsOutput, error)

	// MoveFile changes the file's key, which renames it or moves it to another
	// directory. The file keeps its ID.
	MoveFile(ctx context.Context, input MoveFileInput) (*FileObject, error)

	// DeleteFile removes the file and its metadata.
	DeleteFile(ctx context.Context, input DeleteFileInput) error
}

// InitialiseUploadInput is also the request body sent to the server.
type InitialiseUploadInput struct {
	Key         string `json:"key"`
	ContentType string `json:"content_type"`
}

// MaxPartSizeBytes is the largest part the server accepts.
const MaxPartSizeBytes = 5 * 1024 * 1024 // 5 MB

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

type GetUploadObjectInput struct {
	UploadID string
}

type AbortUploadInput struct {
	UploadID string
}

type DownloadFileInput struct {
	FileID string
	// Range is an optional HTTP Range header value, e.g. "bytes=0-1023".
	Range string
}

type DownloadFileOutput struct {
	ContentType        string
	ContentDisposition string
	// ContentRange is set when a Range was requested, e.g. "bytes 0-1023/4096".
	ContentRange string
	Body         io.ReadCloser
}

type GetFileObjectInput struct {
	FileID string
}

// MoveFileInput is also the request body sent to the server. Key is the new
// relative path within the caller's namespace.
type MoveFileInput struct {
	FileID string `json:"-"`
	Key    string `json:"key"`
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
