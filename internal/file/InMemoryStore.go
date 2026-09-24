package file

import (
	"context"
	"slices"
	"sync"

	"github.com/kduong-dev/goutil/eventsource"
	"github.com/kduong-dev/goutil/eventsource/subscription"
	"github.com/kduong-dev/goutil/fatal"
	"github.com/kduong-dev/storage-service/internal/upload"
)

var _ Store = (*InMemoryStore)(nil)

// InMemoryStore holds uploads and file infos in slices indexed by ID. Every
// change is appended to the event log and applied by catching up on it, so
// the log remains the source of truth and state is rebuilt on restart.
type InMemoryStore struct {
	mutex                 sync.Mutex
	log                   eventsource.Log
	legacyNamespace       string
	cursor                int64
	uploads               []*upload.Object
	uploadIndexByID       map[string]int
	fileInfos             []*Object
	fileInfoIndexByFileID map[string]int
}

type NewInMemoryStoreInput struct {
	Log eventsource.Log
	// LegacyNamespace is assigned to uploads recorded before namespaces existed.
	LegacyNamespace string
}

func NewInMemoryStore(input NewInMemoryStoreInput) *InMemoryStore {
	return &InMemoryStore{
		log:                   input.Log,
		legacyNamespace:       input.LegacyNamespace,
		uploadIndexByID:       make(map[string]int),
		fileInfoIndexByFileID: make(map[string]int),
	}
}

func (store *InMemoryStore) InitialiseUpload(ctx context.Context, object *upload.Object) error {
	store.mutex.Lock()
	defer store.mutex.Unlock()
	return store.appendAndCatchUp(ctx, EventFrame{
		EventBase: eventsource.NewEventBase(EventTypeUploadInitiated),
		UploadInitiatedEvent: &UploadInitiatedEvent{
			UploadID:    object.ID,
			Namespace:   object.Namespace,
			Key:         object.Key,
			ContentType: object.ContentType,
			CreatedAt:   object.CreatedAt,
		},
	})
}

func (store *InMemoryStore) RecordPart(ctx context.Context, uploadID string, part upload.Part, updatedAt string) error {
	store.mutex.Lock()
	defer store.mutex.Unlock()
	if err := store.assertUploadActive(ctx, uploadID); err != nil {
		return err
	}
	return store.appendAndCatchUp(ctx, EventFrame{
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

func (store *InMemoryStore) CompleteUpload(ctx context.Context, input CompleteUploadInput) error {
	store.mutex.Lock()
	defer store.mutex.Unlock()
	if err := store.assertUploadActive(ctx, input.UploadID); err != nil {
		return err
	}
	return store.appendAndCatchUp(ctx, EventFrame{
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

func (store *InMemoryStore) AbortUpload(ctx context.Context, uploadID string, updatedAt string) error {
	store.mutex.Lock()
	defer store.mutex.Unlock()
	if err := store.assertUploadActive(ctx, uploadID); err != nil {
		return err
	}
	return store.appendAndCatchUp(ctx, EventFrame{
		EventBase: eventsource.NewEventBase(EventTypeUploadAborted),
		UploadAbortedEvent: &UploadAbortedEvent{
			UploadID:  uploadID,
			UpdatedAt: updatedAt,
		},
	})
}

// GetUpload returns a copy so callers never observe later changes mid-read.
func (store *InMemoryStore) GetUpload(ctx context.Context, uploadID string) (*upload.Object, error) {
	store.mutex.Lock()
	defer store.mutex.Unlock()
	store.catchUp(ctx)
	object, ok := store.findUpload(uploadID)
	if !ok {
		return nil, ErrUploadNotFound
	}
	copied := *object
	copied.Parts = slices.Clone(object.Parts)
	return &copied, nil
}

func (store *InMemoryStore) GetFileInfo(ctx context.Context, fileID string) (*Object, error) {
	store.mutex.Lock()
	defer store.mutex.Unlock()
	store.catchUp(ctx)
	index, ok := store.fileInfoIndexByFileID[fileID]
	if !ok {
		return nil, ErrFileNotFound
	}
	copied := *store.fileInfos[index]
	return &copied, nil
}

func (store *InMemoryStore) findUpload(uploadID string) (*upload.Object, bool) {
	index, ok := store.uploadIndexByID[uploadID]
	if !ok {
		return nil, false
	}
	return store.uploads[index], true
}

func (store *InMemoryStore) assertUploadActive(ctx context.Context, uploadID string) error {
	store.catchUp(ctx)
	object, ok := store.findUpload(uploadID)
	if !ok {
		return ErrUploadNotFound
	}
	if object.Status != upload.StatusInitiated {
		return ErrUploadNotActive
	}
	return nil
}

func (store *InMemoryStore) appendAndCatchUp(ctx context.Context, frame EventFrame) error {
	if _, err := store.log.Append(fatal.UnlessMarshal(frame)); err != nil {
		return err
	}
	store.catchUp(ctx)
	return nil
}

func (store *InMemoryStore) catchUp(ctx context.Context) {
	var err error
	store.cursor, err = subscription.CatchUp(ctx, subscription.Input{
		Log:    store.log,
		Cursor: store.cursor,
		Apply:  store.apply,
	})
	fatal.OnError(err)
}

func (store *InMemoryStore) apply(ctx context.Context, event *eventsource.Event) error {
	var frame EventFrame
	fatal.UnlessUnmarshal(event.Data, &frame)
	switch frame.Type {
	case EventTypeUploadInitiated:
		store.applyInitiated(frame.UploadInitiatedEvent)
	case EventTypePartUploaded:
		store.applyPartUploaded(frame.PartUploadedEvent)
	case EventTypeUploadCompleted:
		store.applyCompleted(frame.UploadCompletedEvent)
	case EventTypeUploadAborted:
		store.applyAborted(frame.UploadAbortedEvent)
	}
	return nil
}

func (store *InMemoryStore) applyInitiated(event *UploadInitiatedEvent) {
	namespace := event.Namespace
	if namespace == "" {
		// Events written before namespaces existed carried a user id instead.
		namespace = store.legacyNamespace
	}
	store.uploadIndexByID[event.UploadID] = len(store.uploads)
	store.uploads = append(store.uploads, &upload.Object{
		ID:          event.UploadID,
		Namespace:   namespace,
		Key:         event.Key,
		ContentType: event.ContentType,
		Status:      upload.StatusInitiated,
		CreatedAt:   event.CreatedAt,
		UpdatedAt:   event.CreatedAt,
	})
}

func (store *InMemoryStore) applyPartUploaded(event *PartUploadedEvent) {
	object, ok := store.findUpload(event.UploadID)
	if !ok {
		return
	}
	part := upload.Part{Number: event.PartNumber, Size: event.Size, Checksum: event.Checksum}
	object.UpdatedAt = event.UpdatedAt
	for index, existing := range object.Parts {
		if existing.Number == event.PartNumber {
			object.Parts[index] = part
			return
		}
	}
	object.Parts = append(object.Parts, part)
}

func (store *InMemoryStore) applyCompleted(event *UploadCompletedEvent) {
	object, ok := store.findUpload(event.UploadID)
	if !ok {
		return
	}
	object.Status = upload.StatusCompleted
	object.UpdatedAt = event.UpdatedAt
	store.fileInfoIndexByFileID[event.FileID] = len(store.fileInfos)
	store.fileInfos = append(store.fileInfos, &Object{
		ID:          event.FileID,
		Namespace:   object.Namespace,
		UploadID:    event.UploadID,
		Key:         object.Key,
		ContentType: object.ContentType,
		Size:        event.Size,
		Checksum:    event.Checksum,
		CreatedAt:   event.UpdatedAt,
	})
}

func (store *InMemoryStore) applyAborted(event *UploadAbortedEvent) {
	object, ok := store.findUpload(event.UploadID)
	if !ok {
		return
	}
	object.Status = upload.StatusAborted
	object.UpdatedAt = event.UpdatedAt
}
