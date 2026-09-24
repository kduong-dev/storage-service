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
	objectByFileID map[string]*storageservice.File
	objects        SortedObjects
	// revisionByKey is the latest revision given to each key, deleted or not.
	revisionByKey map[string]int
}

type NewEventSourcedObjectStoreInput struct {
	Log eventsource.Log
}

func NewEventSourcedObjectStore(input NewEventSourcedObjectStoreInput) *EventSourcedObjectStore {
	return &EventSourcedObjectStore{
		log:            input.Log,
		objectByFileID: make(map[string]*storageservice.File),
		revisionByKey:  make(map[string]int),
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

func (store *EventSourcedObjectStore) Put(ctx context.Context, object *storageservice.File) (*storageservice.File, error) {
	store.catchUp(ctx)
	if _, ok := store.objectByFileID[object.ID]; ok {
		return nil, ErrAlreadyExists
	}
	copied := *object
	copied.Revision = store.revisionByKey[object.Key] + 1
	if _, err := store.log.Append(fatal.UnlessMarshal(EventFrame{
		EventBase:        eventsource.NewEventBase(EventTypeFileCreated),
		FileCreatedEvent: &copied,
	})); err != nil {
		return nil, err
	}
	return &copied, nil
}

func (store *EventSourcedObjectStore) Get(ctx context.Context, fileID string) (*storageservice.File, error) {
	store.catchUp(ctx)
	object, ok := store.objectByFileID[fileID]
	if !ok {
		return nil, ErrNotFound
	}
	copied := *object
	return &copied, nil
}

func (store *EventSourcedObjectStore) List(ctx context.Context, input ListInput) (*ListOutput, error) {
	fatal.Unless(input.Limit > 0, "list limit must be positive")
	store.catchUp(ctx)
	var after *storageservice.File
	if input.After != "" {
		object, ok := store.objectByFileID[input.After]
		if !ok || !strings.HasPrefix(object.Key, input.KeyPrefix) {
			return nil, ErrInvalidAfter
		}
		after = object
	}
	page, hasMore := store.objects.Page(PageInput{KeyPrefix: input.KeyPrefix, After: after, Limit: input.Limit})
	output := &ListOutput{Objects: make([]*storageservice.File, len(page))}
	for index, object := range page {
		copied := *object
		output.Objects[index] = &copied
	}
	if hasMore {
		output.NextAfter = page[len(page)-1].ID
	}
	return output, nil
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
		object := frame.FileCreatedEvent
		store.objectByFileID[object.ID] = object
		store.objects.Add(object)
		store.revisionByKey[object.Key] = max(store.revisionByKey[object.Key], object.Revision)
	case EventTypeFileDeleted:
		object := store.objectByFileID[frame.FileDeletedEvent.FileID]
		delete(store.objectByFileID, object.ID)
		store.objects.Remove(object)
	}
	return nil
}
