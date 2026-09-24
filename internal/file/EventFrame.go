package file

import "github.com/kduong-dev/goutil/eventsource"

const EventTypeFileCreated eventsource.EventType = "file_created"

type EventFrame struct {
	eventsource.EventBase
	FileCreatedEvent *Object `json:"file_created_event,omitempty"`
}
