package file

import (
	"context"

	"github.com/kduong-dev/goutil/eventsource"
	"github.com/kduong-dev/goutil/fatal"
	"github.com/kduong-dev/storage-service/internal/projection"
)

var _ ObjectStore = (*InMemoryObjectStore)(nil)

type InMemoryObjectStore struct {
	projection    *projection.Projection
	objects       []*Object
	indexByFileID map[string]int
}

type NewInMemoryObjectStoreInput struct {
	Log eventsource.Log
}

func NewInMemoryObjectStore(input NewInMemoryObjectStoreInput) *InMemoryObjectStore {
	store := &InMemoryObjectStore{indexByFileID: make(map[string]int)}
	store.projection = projection.New(projection.NewInput{Log: input.Log, Apply: store.apply})
	return store
}

func (store *InMemoryObjectStore) Put(ctx context.Context, object *Object) error {
	store.projection.CatchUp(ctx)
	if _, ok := store.indexByFileID[object.ID]; ok {
		return ErrAlreadyExists
	}
	copied := *object
	return store.projection.Append(ctx, EventFrame{
		EventBase:        eventsource.NewEventBase(EventTypeFileCreated),
		FileCreatedEvent: &copied,
	})
}

func (store *InMemoryObjectStore) Get(ctx context.Context, fileID string) (*Object, error) {
	store.projection.CatchUp(ctx)
	index, ok := store.indexByFileID[fileID]
	if !ok {
		return nil, ErrNotFound
	}
	copied := *store.objects[index]
	return &copied, nil
}

func (store *InMemoryObjectStore) apply(ctx context.Context, event *eventsource.Event) error {
	var frame EventFrame
	fatal.UnlessUnmarshal(event.Data, &frame)
	if frame.Type == EventTypeFileCreated {
		store.indexByFileID[frame.FileCreatedEvent.ID] = len(store.objects)
		store.objects = append(store.objects, frame.FileCreatedEvent)
	}
	return nil
}
