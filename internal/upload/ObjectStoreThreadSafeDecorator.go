package upload

import (
	"context"
	"sync"

	"github.com/kduong-dev/storage-service/pkg/storageservice"
)

var _ ObjectStore = (*ObjectStoreThreadSafeDecorator)(nil)

type ObjectStoreThreadSafeDecorator struct {
	mutex     sync.Mutex
	decorated ObjectStore
}

type NewObjectStoreThreadSafeDecoratorInput struct {
	Decorated ObjectStore
}

func NewObjectStoreThreadSafeDecorator(input NewObjectStoreThreadSafeDecoratorInput) *ObjectStoreThreadSafeDecorator {
	return &ObjectStoreThreadSafeDecorator{decorated: input.Decorated}
}

func (decorator *ObjectStoreThreadSafeDecorator) Initialise(ctx context.Context, object *storageservice.Upload) error {
	decorator.mutex.Lock()
	defer decorator.mutex.Unlock()
	return decorator.decorated.Initialise(ctx, object)
}

func (decorator *ObjectStoreThreadSafeDecorator) RecordPart(ctx context.Context, input RecordPartInput) error {
	decorator.mutex.Lock()
	defer decorator.mutex.Unlock()
	return decorator.decorated.RecordPart(ctx, input)
}

func (decorator *ObjectStoreThreadSafeDecorator) Complete(ctx context.Context, input CompleteInput) error {
	decorator.mutex.Lock()
	defer decorator.mutex.Unlock()
	return decorator.decorated.Complete(ctx, input)
}

func (decorator *ObjectStoreThreadSafeDecorator) Abort(ctx context.Context, input AbortInput) error {
	decorator.mutex.Lock()
	defer decorator.mutex.Unlock()
	return decorator.decorated.Abort(ctx, input)
}

func (decorator *ObjectStoreThreadSafeDecorator) Get(ctx context.Context, uploadID string) (*storageservice.Upload, error) {
	decorator.mutex.Lock()
	defer decorator.mutex.Unlock()
	return decorator.decorated.Get(ctx, uploadID)
}
