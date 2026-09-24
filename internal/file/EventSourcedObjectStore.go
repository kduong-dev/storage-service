package file

import (
	"cmp"
	"context"
	"slices"
	"strings"

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

func (store *EventSourcedObjectStore) List(ctx context.Context, input ListInput) (*ListOutput, error) {
	fatal.Unless(input.Limit > 0, "list limit must be positive")
	store.catchUp(ctx)
	var after *Object
	if input.After != "" {
		object, ok := store.objectByFileID[input.After]
		if !ok || !strings.HasPrefix(object.Key, input.KeyPrefix) {
			return nil, ErrInvalidAfter
		}
		after = object
	}
	var matches []*Object
	for _, object := range store.objectByFileID {
		if strings.HasPrefix(object.Key, input.KeyPrefix) && (after == nil || compareObjects(object, after) > 0) {
			matches = append(matches, object)
		}
	}
	slices.SortFunc(matches, compareObjects)
	output := &ListOutput{Objects: make([]*Object, 0, min(len(matches), input.Limit))}
	for _, object := range matches[:min(len(matches), input.Limit)] {
		copied := *object
		output.Objects = append(output.Objects, &copied)
	}
	if len(matches) > input.Limit {
		output.NextAfter = output.Objects[len(output.Objects)-1].ID
	}
	return output, nil
}

// compareObjects orders by key, then by ID since several uploads can share a key.
func compareObjects(left *Object, right *Object) int {
	return cmp.Or(strings.Compare(left.Key, right.Key), strings.Compare(left.ID, right.ID))
}

func (store *EventSourcedObjectStore) apply(ctx context.Context, event *eventsource.Event) error {
	var frame EventFrame
	fatal.UnlessUnmarshal(event.Data, &frame)
	if frame.Type == EventTypeFileCreated {
		store.objectByFileID[frame.FileCreatedEvent.ID] = frame.FileCreatedEvent
	}
	return nil
}
