package storageservice

import (
	"context"
	"errors"
	"io"
	"time"

	"github.com/kduong-dev/goutil/config"
	"github.com/kduong-dev/goutil/fatal"
)

var (
	ErrUnauthorized   = errors.New("unauthorized")
	ErrUploadNotFound = errors.New("upload not found")
	ErrFileNotFound   = errors.New("file not found")
	ErrBadRequest     = errors.New("bad request")
	ErrServerError    = errors.New("server error")
)

// Upload tracks a multipart upload session.
type Upload struct {
	ID          string `json:"id"`
	Key         string `json:"key"`
	ContentType string `json:"content_type"`
	Parts       []Part `json:"parts,omitempty"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

// Part describes one uploaded chunk.
type Part struct {
	PartNumber int    `json:"part_number"`
	Size       int64  `json:"size"`
	Checksum   string `json:"checksum"`
}

// File is the completed, stored object produced after an upload is finalised.
type File struct {
	ID          string `json:"id"`
	UploadID    string `json:"upload_id"`
	Key         string `json:"key"`
	ContentType string `json:"content_type"`
	Size        int64  `json:"size"`
	Checksum    string `json:"checksum"`
	CreatedAt   string `json:"created_at"`
}

// UploadPartResponse holds metadata returned after a part is accepted.
type UploadPartResponse struct {
	PartNumber int    `json:"part_number"`
	Size       int64  `json:"size"`
	Checksum   string `json:"checksum"`
}

// DownloadFileResponse holds the streamed file content and its metadata.
type DownloadFileResponse struct {
	ContentType        string
	ContentDisposition string
	Body               io.ReadCloser
}

// Client is the public interface for the storage-service API. Every call is
// scoped to the namespace of the API key the client was configured with;
// deciding which end user may access a file is the calling service's job.
type Client interface {
	// InitialiseUpload begins a new multipart upload session. key is a
	// relative path within the caller's namespace.
	InitialiseUpload(ctx context.Context, key string, contentType string) (*Upload, error)

	// UploadPart streams one chunk to an existing upload session.
	UploadPart(ctx context.Context, uploadID string, partNumber int, body io.Reader) (*UploadPartResponse, error)

	// CompleteUpload finalises an upload session and assembles all parts into a File.
	CompleteUpload(ctx context.Context, uploadID string) (*File, error)

	// AbortUpload cancels an upload session and discards its uploaded parts.
	AbortUpload(ctx context.Context, uploadID string) error

	// DownloadFile streams the assembled file for the given file ID.
	// The caller is responsible for closing DownloadFileResponse.Body.
	DownloadFile(ctx context.Context, fileID string) (*DownloadFileResponse, error)

	// ListFiles returns one page of the caller's files, ordered by key. Pass
	// the returned NextCursor back as Cursor to fetch the next page.
	ListFiles(ctx context.Context, input ListFilesInput) (*ListFilesResponse, error)
}

type ListFilesInput struct {
	// Prefix filters to keys starting with it, relative to the caller's namespace.
	Prefix string
	Cursor string
	// Limit is the page size; zero uses the server default of 100, at most 1000.
	Limit int
}

type ListFilesResponse struct {
	Files []*File `json:"files"`
	// NextCursor is empty on the last page.
	NextCursor string `json:"next_cursor,omitempty"`
}

func ClientFromEnv() Client {
	implementation := config.EnvStringOrFatal("STORAGE_SERVICE_CLIENT_IMPLEMENTATION")
	switch implementation {
	case "HTTP":
		return NewHTTPClient(NewHTTPClientInput{
			Timeout: config.EnvDuration("STORAGE_SERVICE_HTTP_CLIENT_TIMEOUT", 20*time.Second),
			BaseURL: *config.EnvURLOrFatal("STORAGE_SERVICE_URL"),
			APIKey:  config.EnvStringOrFatal("STORAGE_SERVICE_API_KEY"),
		})
	default:
		fatal.LogErrorf("invalid storage service client implementation: %s", implementation)
		return nil
	}
}
