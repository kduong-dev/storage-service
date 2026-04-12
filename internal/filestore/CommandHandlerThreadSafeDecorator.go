package filestore

import (
	"context"
	"sync"
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

func (decorator *CommandHandlerThreadSafeDecorator) InitialiseUpload(ctx context.Context, upload *Upload) error {
	decorator.mutex.Lock()
	defer decorator.mutex.Unlock()
	return decorator.decorated.InitialiseUpload(ctx, upload)
}

func (decorator *CommandHandlerThreadSafeDecorator) RecordPart(ctx context.Context, uploadID string, part Part, updatedAt string) error {
	decorator.mutex.Lock()
	defer decorator.mutex.Unlock()
	return decorator.decorated.RecordPart(ctx, uploadID, part, updatedAt)
}

func (decorator *CommandHandlerThreadSafeDecorator) CompleteUpload(ctx context.Context, uploadID string, fileID string, size int64, checksum string, updatedAt string) error {
	decorator.mutex.Lock()
	defer decorator.mutex.Unlock()
	return decorator.decorated.CompleteUpload(ctx, uploadID, fileID, size, checksum, updatedAt)
}

func (decorator *CommandHandlerThreadSafeDecorator) AbortUpload(ctx context.Context, uploadID string, updatedAt string) error {
	decorator.mutex.Lock()
	defer decorator.mutex.Unlock()
	return decorator.decorated.AbortUpload(ctx, uploadID, updatedAt)
}
