package filestore

import (
	"context"
	"sync"
)

var _ QueryHandler = (*QueryHandlerThreadSafeDecorator)(nil)

type QueryHandlerThreadSafeDecorator struct {
	mutex     sync.Mutex
	decorated QueryHandler
}

type NewQueryHandlerThreadSafeDecoratorInput struct {
	Decorated QueryHandler
}

func NewQueryHandlerThreadSafeDecorator(input NewQueryHandlerThreadSafeDecoratorInput) *QueryHandlerThreadSafeDecorator {
	return &QueryHandlerThreadSafeDecorator{decorated: input.Decorated}
}

func (d *QueryHandlerThreadSafeDecorator) GetUpload(ctx context.Context, uploadID string) (*Upload, error) {
	d.mutex.Lock()
	defer d.mutex.Unlock()
	return d.decorated.GetUpload(ctx, uploadID)
}

func (d *QueryHandlerThreadSafeDecorator) GetFile(ctx context.Context, fileID string) (*File, error) {
	d.mutex.Lock()
	defer d.mutex.Unlock()
	return d.decorated.GetFile(ctx, fileID)
}
