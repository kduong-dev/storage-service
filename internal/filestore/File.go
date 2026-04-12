package filestore

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

// File is the completed, stored object produced after an upload is finalised.
type File struct {
	ID          string `json:"id"`
	UserID      string `json:"user_id"`
	UploadID    string `json:"upload_id"`
	Filename    string `json:"filename"`
	ContentType string `json:"content_type"`
	Size        int64  `json:"size"`
	Checksum    string `json:"checksum"` // hex-encoded MD5 of the full object
	CreatedAt   string `json:"created_at"`
}
