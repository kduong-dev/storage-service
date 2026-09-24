package upload

import (
	"context"
	"slices"

	"github.com/kduong-dev/goutil/eventsource"
	"github.com/kduong-dev/goutil/eventsource/subscription"
	"github.com/kduong-dev/goutil/fatal"
	"github.com/kduong-dev/storage-service/pkg/storageservice"
)

var _ ObjectStore = (*EventSourcedObjectStore)(nil)

type EventSourcedObjectStore struct {
	log              eventsource.Log
	cursor           int64
	objectByUploadID map[string]*storageservice.Upload
}

type NewEventSourcedObjectStoreInput struct {
	Log eventsource.Log
}

func NewEventSourcedObjectStore(input NewEventSourcedObjectStoreInput) *EventSourcedObjectStore {
	return &EventSourcedObjectStore{
		log:              input.Log,
		objectByUploadID: make(map[string]*storageservice.Upload),
	}
}

func (store *EventSourcedObjectStore) catchUp(ctx context.Context) {
	var err error
	store.cursor, err = subscription.CatchUp(ctx, subscription.Input{
		Log:    store.log,
		Cursor: store.cursor,
		Apply:  store.apply,
	})
	fatal.OnError(err)
}

func (store *EventSourcedObjectStore) assertExists(ctx context.Context, uploadID string) error {
	store.catchUp(ctx)
	if _, ok := store.objectByUploadID[uploadID]; !ok {
		return ErrNotFound
	}
	return nil
}

func (store *EventSourcedObjectStore) append(frame EventFrame) error {
	_, err := store.log.Append(fatal.UnlessMarshal(frame))
	return err
}

func (store *EventSourcedObjectStore) Initialise(ctx context.Context, object *storageservice.Upload) error {
	return store.append(EventFrame{
		EventBase: eventsource.NewEventBase(EventTypeUploadInitiated),
		UploadInitiatedEvent: &UploadInitiatedEvent{
			UploadID:    object.ID,
			Key:         object.Key,
			ContentType: object.ContentType,
			CreatedAt:   object.CreatedAt,
		},
	})
}

func (store *EventSourcedObjectStore) RecordPart(ctx context.Context, input RecordPartInput) error {
	if err := store.assertExists(ctx, input.UploadID); err != nil {
		return err
	}
	return store.append(EventFrame{
		EventBase: eventsource.NewEventBase(EventTypePartUploaded),
		PartUploadedEvent: &PartUploadedEvent{
			UploadID:   input.UploadID,
			PartNumber: input.Part.PartNumber,
			Size:       input.Part.Size,
			Checksum:   input.Part.Checksum,
			UpdatedAt:  input.UpdatedAt,
		},
	})
}

func (store *EventSourcedObjectStore) Complete(ctx context.Context, input CompleteInput) error {
	if err := store.assertExists(ctx, input.UploadID); err != nil {
		return err
	}
	return store.append(EventFrame{
		EventBase: eventsource.NewEventBase(EventTypeUploadCompleted),
		UploadCompletedEvent: &UploadCompletedEvent{
			UploadID:  input.UploadID,
			FileID:    input.FileID,
			Size:      input.Size,
			Checksum:  input.Checksum,
			UpdatedAt: input.UpdatedAt,
		},
	})
}

func (store *EventSourcedObjectStore) Abort(ctx context.Context, input AbortInput) error {
	if err := store.assertExists(ctx, input.UploadID); err != nil {
		return err
	}
	return store.append(EventFrame{
		EventBase: eventsource.NewEventBase(EventTypeUploadAborted),
		UploadAbortedEvent: &UploadAbortedEvent{
			UploadID:  input.UploadID,
			UpdatedAt: input.UpdatedAt,
		},
	})
}

func (store *EventSourcedObjectStore) Get(ctx context.Context, uploadID string) (*storageservice.Upload, error) {
	store.catchUp(ctx)
	object, ok := store.objectByUploadID[uploadID]
	if !ok {
		return nil, ErrNotFound
	}
	copied := *object
	copied.Parts = slices.Clone(object.Parts)
	return &copied, nil
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
		delete(store.objectByUploadID, frame.UploadCompletedEvent.UploadID)
	case EventTypeUploadAborted:
		delete(store.objectByUploadID, frame.UploadAbortedEvent.UploadID)
	}
	return nil
}

func (store *EventSourcedObjectStore) applyInitiated(event *UploadInitiatedEvent) {
	store.objectByUploadID[event.UploadID] = &storageservice.Upload{
		ID:          event.UploadID,
		Key:         event.Key,
		ContentType: event.ContentType,
		CreatedAt:   event.CreatedAt,
		UpdatedAt:   event.CreatedAt,
	}
}

func (store *EventSourcedObjectStore) applyPartUploaded(event *PartUploadedEvent) {
	object, ok := store.objectByUploadID[event.UploadID]
	if !ok {
		return
	}
	part := storageservice.Part{PartNumber: event.PartNumber, Size: event.Size, Checksum: event.Checksum}
	object.UpdatedAt = event.UpdatedAt
	for index, existing := range object.Parts {
		if existing.PartNumber == event.PartNumber {
			object.Parts[index] = part
			return
		}
	}
	object.Parts = append(object.Parts, part)
}
