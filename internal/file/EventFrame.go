package file

import (
	"github.com/kduong-dev/goutil/eventsource"
	"github.com/kduong-dev/storage-service/pkg/storageservice"
)

const (
	EventTypeFileCreated eventsource.EventType = "file_created"
	EventTypeFileMoved   eventsource.EventType = "file_moved"
	EventTypeFileDeleted eventsource.EventType = "file_deleted"
)

type EventFrame struct {
	eventsource.EventBase
	FileCreatedEvent *storageservice.FileObject `json:"file_created_event,omitempty"`
	FileMovedEvent   *FileMovedEvent            `json:"file_moved_event,omitempty"`
	FileDeletedEvent *FileDeletedEvent          `json:"file_deleted_event,omitempty"`
}

type FileMovedEvent struct {
	FileID string `json:"file_id"`
	Key    string `json:"key"`
}

type FileDeletedEvent struct {
	FileID string `json:"file_id"`
}
