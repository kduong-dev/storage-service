package file

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

func (decorator *ObjectStoreThreadSafeDecorator) Put(ctx context.Context, object *storageservice.FileObject) error {
	decorator.mutex.Lock()
	defer decorator.mutex.Unlock()
	return decorator.decorated.Put(ctx, object)
}

func (decorator *ObjectStoreThreadSafeDecorator) Get(ctx context.Context, fileID string) (*storageservice.FileObject, error) {
	decorator.mutex.Lock()
	defer decorator.mutex.Unlock()
	return decorator.decorated.Get(ctx, fileID)
}

func (decorator *ObjectStoreThreadSafeDecorator) List(ctx context.Context, input ListInput) (*storageservice.ListFileObjectsOutput, error) {
	decorator.mutex.Lock()
	defer decorator.mutex.Unlock()
	return decorator.decorated.List(ctx, input)
}

func (decorator *ObjectStoreThreadSafeDecorator) Delete(ctx context.Context, fileID string) error {
	decorator.mutex.Lock()
	defer decorator.mutex.Unlock()
	return decorator.decorated.Delete(ctx, fileID)
}
