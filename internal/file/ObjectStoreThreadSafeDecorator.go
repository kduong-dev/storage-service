package file

import (
	"context"
	"sync"
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

func (decorator *ObjectStoreThreadSafeDecorator) Put(ctx context.Context, object *Object) error {
	decorator.mutex.Lock()
	defer decorator.mutex.Unlock()
	return decorator.decorated.Put(ctx, object)
}

func (decorator *ObjectStoreThreadSafeDecorator) Get(ctx context.Context, fileID string) (*Object, error) {
	decorator.mutex.Lock()
	defer decorator.mutex.Unlock()
	return decorator.decorated.Get(ctx, fileID)
}
