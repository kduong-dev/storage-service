package storageservice

import (
	"context"
	"errors"
	"io"
	"time"

	"github.com/kduong/trading-backend/internal/config"
)

var (
	ErrUploadNotFound  = errors.New("upload not found")
	ErrUploadForbidden = errors.New("upload forbidden")
	ErrFileNotFound    = errors.New("file not found")
	ErrFileForbidden   = errors.New("file forbidden")
	ErrUploadNotActive = errors.New("upload is not active")
	ErrServerError     = errors.New("server error")
)

// UploadStatus represents the state of a multipart upload.
type UploadStatus string

const (
	UploadStatusInitiated UploadStatus = "initiated"
	UploadStatusCompleted UploadStatus = "completed"
	UploadStatusAborted   UploadStatus = "aborted"
)

// Upload tracks a multipart upload session.
type Upload struct {
	ID          string       `json:"id"`
	UserID      string       `json:"user_id"`
	Filename    string       `json:"filename"`
	ContentType string       `json:"content_type"`
	Status      UploadStatus `json:"status"`
	Parts       []Part       `json:"parts,omitempty"`
	CreatedAt   string       `json:"created_at"`
	UpdatedAt   string       `json:"updated_at"`
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
	UserID      string `json:"user_id"`
	UploadID    string `json:"upload_id"`
	Filename    string `json:"filename"`
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

// Client is the public interface for the storage-service API.
type Client interface {
	// InitialiseUpload begins a new multipart upload session.
	InitialiseUpload(ctx context.Context, filename string, contentType string) (*Upload, error)

	// UploadPart streams one chunk to an existing upload session.
	UploadPart(ctx context.Context, uploadID string, partNumber int, body io.Reader) (*UploadPartResponse, error)

	// CompleteUpload finalises an upload session and assembles all parts into a File.
	CompleteUpload(ctx context.Context, uploadID string) (*File, error)

	// DownloadFile streams the assembled file for the given file ID.
	// The caller is responsible for closing DownloadFileResponse.Body.
	DownloadFile(ctx context.Context, fileID string) (*DownloadFileResponse, error)
}

func ClientFromEnv() Client {
	implementation := config.EnvStringOrFatal("STORAGE_SERVICE_CLIENT_IMPLEMENTATION")
	switch implementation {
	case "HTTP":
		return NewHTTPClient(NewHTTPClientInput{
			Timeout: config.EnvDuration("STORAGE_SERVICE_HTTP_CLIENT_TIMEOUT", 20*time.Second),
			BaseURL: config.EnvURLOrFatal("STORAGE_SERVICE"),
		})
	default:
		panic("invalid storage service client implementation: " + implementation)
	}
}
