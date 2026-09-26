package file

import (
	"context"
	"strings"

	"github.com/kduong-dev/goutil/eventsource"
	"github.com/kduong-dev/goutil/eventsource/subscription"
	"github.com/kduong-dev/goutil/fatal"
	"github.com/kduong-dev/storage-service/pkg/storageservice"
)

var _ ObjectStore = (*EventSourcedObjectStore)(nil)

type EventSourcedObjectStore struct {
	log            eventsource.Log
	cursor         int64
	objectByFileID map[string]*storageservice.FileObject
	objects        SortedObjects
}

type NewEventSourcedObjectStoreInput struct {
	Log eventsource.Log
}

func NewEventSourcedObjectStore(input NewEventSourcedObjectStoreInput) *EventSourcedObjectStore {
	return &EventSourcedObjectStore{
		log:            input.Log,
		objectByFileID: make(map[string]*storageservice.FileObject),
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

func (store *EventSourcedObjectStore) Put(ctx context.Context, object *storageservice.FileObject) error {
	store.catchUp(ctx)
	if _, ok := store.objectByFileID[object.ID]; ok {
		return ErrAlreadyExists
	}
	copied := *object
	_, err := store.log.Append(fatal.UnlessMarshal(EventFrame{
		EventBase:        eventsource.NewEventBase(EventTypeFileCreated),
		FileCreatedEvent: &copied,
	}))
	return err
}

func (store *EventSourcedObjectStore) Get(ctx context.Context, fileID string) (*storageservice.FileObject, error) {
	store.catchUp(ctx)
	object, ok := store.objectByFileID[fileID]
	if !ok {
		return nil, ErrNotFound
	}
	copied := *object
	return &copied, nil
}

func (store *EventSourcedObjectStore) List(ctx context.Context, input ListInput) (*storageservice.ListFileObjectsOutput, error) {
	fatal.Unless(input.Limit > 0, "list limit must be positive")
	store.catchUp(ctx)
	var after *storageservice.FileObject
	if input.After != "" {
		object, ok := store.objectByFileID[input.After]
		if !ok || !strings.HasPrefix(object.Key, input.KeyPrefix) {
			return nil, ErrInvalidAfter
		}
		after = object
	}
	page, hasMore := store.objects.Page(PageInput{KeyPrefix: input.KeyPrefix, After: after, Limit: input.Limit})
	output := &storageservice.ListFileObjectsOutput{Files: make([]*storageservice.FileObject, len(page))}
	for index, object := range page {
		copied := *object
		output.Files[index] = &copied
	}
	if hasMore {
		output.NextCursor = page[len(page)-1].ID
	}
	return output, nil
}

func (store *EventSourcedObjectStore) Move(ctx context.Context, input MoveInput) error {
	store.catchUp(ctx)
	if _, ok := store.objectByFileID[input.FileID]; !ok {
		return ErrNotFound
	}
	_, err := store.log.Append(fatal.UnlessMarshal(EventFrame{
		EventBase:      eventsource.NewEventBase(EventTypeFileMoved),
		FileMovedEvent: &FileMovedEvent{FileID: input.FileID, Key: input.Key},
	}))
	return err
}

func (store *EventSourcedObjectStore) Delete(ctx context.Context, fileID string) error {
	store.catchUp(ctx)
	if _, ok := store.objectByFileID[fileID]; !ok {
		return ErrNotFound
	}
	_, err := store.log.Append(fatal.UnlessMarshal(EventFrame{
		EventBase:        eventsource.NewEventBase(EventTypeFileDeleted),
		FileDeletedEvent: &FileDeletedEvent{FileID: fileID},
	}))
	return err
}

func (store *EventSourcedObjectStore) apply(ctx context.Context, event *eventsource.Event) error {
	var frame EventFrame
	fatal.UnlessUnmarshal(event.Data, &frame)
	switch frame.Type {
	case EventTypeFileCreated:
		store.objectByFileID[frame.FileCreatedEvent.ID] = frame.FileCreatedEvent
		store.objects.Add(frame.FileCreatedEvent)
	case EventTypeFileMoved:
		object := store.objectByFileID[frame.FileMovedEvent.FileID]
		// Remove before changing the key, since the sorted position depends on it.
		store.objects.Remove(object)
		object.Key = frame.FileMovedEvent.Key
		store.objects.Add(object)
	case EventTypeFileDeleted:
		object := store.objectByFileID[frame.FileDeletedEvent.FileID]
		delete(store.objectByFileID, object.ID)
		store.objects.Remove(object)
	}
	return nil
}
