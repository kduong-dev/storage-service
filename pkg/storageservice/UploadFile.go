package storageservice

import (
	"bytes"
	"context"
	"io"
)

const partSizeBytes = 5 * 1024 * 1024 // 5 MB

// UploadFileInput holds the parameters for UploadFile.
type UploadFileInput struct {
	Filename    string
	ContentType string
	Body        io.Reader
}

// UploadFile is a helper that performs a full multipart upload in one call.
// It splits the body into parts of up to 5 MB each, uploading them sequentially,
// then completes the upload and returns the resulting File.
func UploadFile(ctx context.Context, client Client, input UploadFileInput) (*File, error) {
	upload, err := client.InitialiseUpload(ctx, input.Filename, input.ContentType)
	if err != nil {
		return nil, err
	}
	partNumber := 1
	buffer := make([]byte, partSizeBytes)
	for {
		bytesRead, readErr := io.ReadFull(input.Body, buffer)
		if bytesRead > 0 {
			_, err = client.UploadPart(ctx, upload.ID, partNumber, bytes.NewReader(buffer[:bytesRead]))
			if err != nil {
				return nil, err
			}
			partNumber++
		}
		if readErr == io.EOF || readErr == io.ErrUnexpectedEOF {
			break
		}
		if readErr != nil {
			return nil, readErr
		}
	}
	return client.CompleteUpload(ctx, upload.ID)
}
