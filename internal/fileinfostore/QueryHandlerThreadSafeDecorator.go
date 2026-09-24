package fileinfostore

import (
	"context"
	"sync"

	"github.com/kduong-dev/storage-service/internal/fileinfo"
	"github.com/kduong-dev/storage-service/internal/upload"
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

func (decorator *QueryHandlerThreadSafeDecorator) GetUpload(ctx context.Context, uploadID string) (*upload.Object, error) {
	decorator.mutex.Lock()
	defer decorator.mutex.Unlock()
	return decorator.decorated.GetUpload(ctx, uploadID)
}

func (decorator *QueryHandlerThreadSafeDecorator) GetFileInfo(ctx context.Context, fileID string) (*fileinfo.FileInfo, error) {
	decorator.mutex.Lock()
	defer decorator.mutex.Unlock()
	return decorator.decorated.GetFileInfo(ctx, fileID)
}
