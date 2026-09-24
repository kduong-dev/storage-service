package file

import (
	"github.com/kduong-dev/goutil/eventsource"
	"github.com/kduong-dev/storage-service/pkg/storageservice"
)

const EventTypeFileCreated eventsource.EventType = "file_created"

type EventFrame struct {
	eventsource.EventBase
	FileCreatedEvent *storageservice.File `json:"file_created_event,omitempty"`
}
