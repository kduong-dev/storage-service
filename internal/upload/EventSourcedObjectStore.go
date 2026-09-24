package upload

import (
	"context"
	"slices"

	"github.com/kduong-dev/goutil/eventsource"
	"github.com/kduong-dev/goutil/fatal"
	"github.com/kduong-dev/storage-service/internal/projection"
)

var _ ObjectStore = (*EventSourcedObjectStore)(nil)

type EventSourcedObjectStore struct {
	projection      *projection.Projection
	objects         []*Object
	indexByUploadID map[string]int
}

type NewEventSourcedObjectStoreInput struct {
	Log eventsource.Log
}

func NewEventSourcedObjectStore(input NewEventSourcedObjectStoreInput) *EventSourcedObjectStore {
	store := &EventSourcedObjectStore{indexByUploadID: make(map[string]int)}
	store.projection = projection.New(projection.NewInput{Log: input.Log, Apply: store.apply})
	return store
}

func (store *EventSourcedObjectStore) Initialise(ctx context.Context, object *Object) error {
	store.projection.CatchUp(ctx)
	if _, ok := store.find(object.ID); ok {
		return ErrAlreadyExists
	}
	return store.projection.Append(ctx, EventFrame{
		EventBase: eventsource.NewEventBase(EventTypeUploadInitiated),
		UploadInitiatedEvent: &UploadInitiatedEvent{
			UploadID:    object.ID,
			Key:         object.Key,
			ContentType: object.ContentType,
			CreatedAt:   object.CreatedAt,
		},
	})
}

func (store *EventSourcedObjectStore) RecordPart(ctx context.Context, uploadID string, part Part, updatedAt string) error {
	if err := store.assertActive(ctx, uploadID); err != nil {
		return err
	}
	return store.projection.Append(ctx, EventFrame{
		EventBase: eventsource.NewEventBase(EventTypePartUploaded),
		PartUploadedEvent: &PartUploadedEvent{
			UploadID:   uploadID,
			PartNumber: part.Number,
			Size:       part.Size,
			Checksum:   part.Checksum,
			UpdatedAt:  updatedAt,
		},
	})
}

func (store *EventSourcedObjectStore) Complete(ctx context.Context, uploadID string, updatedAt string) error {
	if err := store.assertActive(ctx, uploadID); err != nil {
		return err
	}
	return store.projection.Append(ctx, EventFrame{
		EventBase:            eventsource.NewEventBase(EventTypeUploadCompleted),
		UploadCompletedEvent: &UploadStatusEvent{UploadID: uploadID, UpdatedAt: updatedAt},
	})
}

func (store *EventSourcedObjectStore) Abort(ctx context.Context, uploadID string, updatedAt string) error {
	if err := store.assertActive(ctx, uploadID); err != nil {
		return err
	}
	return store.projection.Append(ctx, EventFrame{
		EventBase:          eventsource.NewEventBase(EventTypeUploadAborted),
		UploadAbortedEvent: &UploadStatusEvent{UploadID: uploadID, UpdatedAt: updatedAt},
	})
}

func (store *EventSourcedObjectStore) Get(ctx context.Context, uploadID string) (*Object, error) {
	store.projection.CatchUp(ctx)
	object, ok := store.find(uploadID)
	if !ok {
		return nil, ErrNotFound
	}
	copied := *object
	copied.Parts = slices.Clone(object.Parts)
	return &copied, nil
}

func (store *EventSourcedObjectStore) find(uploadID string) (*Object, bool) {
	index, ok := store.indexByUploadID[uploadID]
	if !ok {
		return nil, false
	}
	return store.objects[index], true
}

func (store *EventSourcedObjectStore) assertActive(ctx context.Context, uploadID string) error {
	store.projection.CatchUp(ctx)
	object, ok := store.find(uploadID)
	if !ok {
		return ErrNotFound
	}
	if object.Status != StatusInitiated {
		return ErrNotActive
	}
	return nil
}

func (store *EventSourcedObjectStore) apply(ctx context.Context, event *eventsource.Event) error {
	var frame EventFrame
	fatal.UnlessUnmarshal(event.Data, &frame)
	switch frame.Type {
	case EventTypeUploadInitiated:
		store.applyInitiated(frame.UploadInitiatedEvent)
	case EventTypePartUploaded:
		store.applyPartUploaded(frame.PartUploadedEvent)
	case EventTypeUploadCompleted:
		store.applyStatus(frame.UploadCompletedEvent, StatusCompleted)
	case EventTypeUploadAborted:
		store.applyStatus(frame.UploadAbortedEvent, StatusAborted)
	}
	return nil
}

func (store *EventSourcedObjectStore) applyInitiated(event *UploadInitiatedEvent) {
	store.indexByUploadID[event.UploadID] = len(store.objects)
	store.objects = append(store.objects, &Object{
		ID:          event.UploadID,
		Key:         event.Key,
		ContentType: event.ContentType,
		Status:      StatusInitiated,
		CreatedAt:   event.CreatedAt,
		UpdatedAt:   event.CreatedAt,
	})
}

func (store *EventSourcedObjectStore) applyPartUploaded(event *PartUploadedEvent) {
	object, ok := store.find(event.UploadID)
	if !ok {
		return
	}
	part := Part{Number: event.PartNumber, Size: event.Size, Checksum: event.Checksum}
	object.UpdatedAt = event.UpdatedAt
	for index, existing := range object.Parts {
		if existing.Number == event.PartNumber {
			object.Parts[index] = part
			return
		}
	}
	object.Parts = append(object.Parts, part)
}

func (store *EventSourcedObjectStore) applyStatus(event *UploadStatusEvent, status Status) {
	object, ok := store.find(event.UploadID)
	if !ok {
		return
	}
	object.Status = status
	object.UpdatedAt = event.UpdatedAt
}
