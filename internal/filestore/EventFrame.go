package filestore

import "github.com/kduong/trading-backend/internal/eventsource"

const (
	EventTypeUploadInitiated  eventsource.EventType = "upload_initiated"
	EventTypePartUploaded     eventsource.EventType = "part_uploaded"
	EventTypeUploadCompleted  eventsource.EventType = "upload_completed"
	EventTypeUploadAborted    eventsource.EventType = "upload_aborted"
)

// EventFrame is the envelope written to the event log.
type EventFrame struct {
	eventsource.EventBase
	UploadInitiatedEvent *UploadInitiatedEvent `json:"upload_initiated_event,omitempty"`
	PartUploadedEvent    *PartUploadedEvent    `json:"part_uploaded_event,omitempty"`
	UploadCompletedEvent *UploadCompletedEvent `json:"upload_completed_event,omitempty"`
	UploadAbortedEvent   *UploadAbortedEvent   `json:"upload_aborted_event,omitempty"`
}

type UploadInitiatedEvent struct {
	UploadID    string `json:"upload_id"`
	UserID      string `json:"user_id"`
	Filename    string `json:"filename"`
	ContentType string `json:"content_type"`
	CreatedAt   string `json:"created_at"`
}

type PartUploadedEvent struct {
	UploadID   string `json:"upload_id"`
	PartNumber int    `json:"part_number"`
	Size       int64  `json:"size"`
	Checksum   string `json:"checksum"`
	UpdatedAt  string `json:"updated_at"`
}

type UploadCompletedEvent struct {
	UploadID  string `json:"upload_id"`
	FileID    string `json:"file_id"`
	Size      int64  `json:"size"`
	Checksum  string `json:"checksum"`
	UpdatedAt string `json:"updated_at"`
}

type UploadAbortedEvent struct {
	UploadID  string `json:"upload_id"`
	UpdatedAt string `json:"updated_at"`
}
