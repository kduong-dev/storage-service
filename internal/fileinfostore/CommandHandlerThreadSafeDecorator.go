package fileinfostore

import (
	"context"
	"sync"

	"github.com/kduong-dev/storage-service/internal/upload"
)

var _ CommandHandler = (*CommandHandlerThreadSafeDecorator)(nil)

type CommandHandlerThreadSafeDecorator struct {
	mutex     sync.Mutex
	decorated CommandHandler
}

type NewCommandHandlerThreadSafeDecoratorInput struct {
	Decorated CommandHandler
}

func NewCommandHandlerThreadSafeDecorator(input NewCommandHandlerThreadSafeDecoratorInput) *CommandHandlerThreadSafeDecorator {
	return &CommandHandlerThreadSafeDecorator{decorated: input.Decorated}
}

func (decorator *CommandHandlerThreadSafeDecorator) InitialiseUpload(ctx context.Context, object *upload.Object) error {
	decorator.mutex.Lock()
	defer decorator.mutex.Unlock()
	return decorator.decorated.InitialiseUpload(ctx, object)
}

func (decorator *CommandHandlerThreadSafeDecorator) RecordPart(ctx context.Context, uploadID string, part upload.Part, updatedAt string) error {
	decorator.mutex.Lock()
	defer decorator.mutex.Unlock()
	return decorator.decorated.RecordPart(ctx, uploadID, part, updatedAt)
}

func (decorator *CommandHandlerThreadSafeDecorator) CompleteUpload(ctx context.Context, input CompleteUploadInput) error {
	decorator.mutex.Lock()
	defer decorator.mutex.Unlock()
	return decorator.decorated.CompleteUpload(ctx, input)
}

func (decorator *CommandHandlerThreadSafeDecorator) AbortUpload(ctx context.Context, uploadID string, updatedAt string) error {
	decorator.mutex.Lock()
	defer decorator.mutex.Unlock()
	return decorator.decorated.AbortUpload(ctx, uploadID, updatedAt)
}
