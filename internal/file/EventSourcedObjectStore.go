package file

import (
	"context"

	"github.com/kduong-dev/goutil/eventsource"
	"github.com/kduong-dev/goutil/eventsource/subscription"
	"github.com/kduong-dev/goutil/fatal"
)

var _ ObjectStore = (*EventSourcedObjectStore)(nil)

type EventSourcedObjectStore struct {
	log            eventsource.Log
	cursor         int64
	objectByFileID map[string]*Object
}

type NewEventSourcedObjectStoreInput struct {
	Log eventsource.Log
}

func NewEventSourcedObjectStore(input NewEventSourcedObjectStoreInput) *EventSourcedObjectStore {
	return &EventSourcedObjectStore{
		log:            input.Log,
		objectByFileID: make(map[string]*Object),
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

func (store *EventSourcedObjectStore) Put(ctx context.Context, object *Object) error {
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

func (store *EventSourcedObjectStore) Get(ctx context.Context, fileID string) (*Object, error) {
	store.catchUp(ctx)
	object, ok := store.objectByFileID[fileID]
	if !ok {
		return nil, ErrNotFound
	}
	copied := *object
	return &copied, nil
}

func (store *EventSourcedObjectStore) apply(ctx context.Context, event *eventsource.Event) error {
	var frame EventFrame
	fatal.UnlessUnmarshal(event.Data, &frame)
	if frame.Type == EventTypeFileCreated {
		store.objectByFileID[frame.FileCreatedEvent.ID] = frame.FileCreatedEvent
	}
	return nil
}
