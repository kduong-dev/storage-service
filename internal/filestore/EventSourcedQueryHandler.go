package filestore

import (
	"context"

	"github.com/kduong-dev/goutil/eventsource"
)

var _ QueryHandler = (*EventSourcedQueryHandler)(nil)

type EventSourcedQueryHandler struct {
	projection *projection
}

type NewEventSourcedQueryHandlerInput struct {
	Log eventsource.Log
	// LegacyNamespace is assigned to uploads recorded before namespaces existed.
	LegacyNamespace string
}

func NewEventSourcedQueryHandler(input NewEventSourcedQueryHandlerInput) *EventSourcedQueryHandler {
	return &EventSourcedQueryHandler{projection: newProjection(input.Log, input.LegacyNamespace)}
}

func (handler *EventSourcedQueryHandler) GetUpload(ctx context.Context, uploadID string) (*Upload, error) {
	handler.projection.catchUp(ctx)
	upload, ok := handler.projection.uploadByID[uploadID]
	if !ok {
		return nil, ErrUploadNotFound
	}
	return upload, nil
}

func (handler *EventSourcedQueryHandler) GetFile(ctx context.Context, fileID string) (*File, error) {
	handler.projection.catchUp(ctx)
	file, ok := handler.projection.fileByID[fileID]
	if !ok {
		return nil, ErrFileNotFound
	}
	return file, nil
}
