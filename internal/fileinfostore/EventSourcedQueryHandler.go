package fileinfostore

import (
	"context"

	"github.com/kduong-dev/goutil/eventsource"
	"github.com/kduong-dev/storage-service/internal/fileinfo"
	"github.com/kduong-dev/storage-service/internal/projection"
	"github.com/kduong-dev/storage-service/internal/upload"
)

var _ QueryHandler = (*EventSourcedQueryHandler)(nil)

type EventSourcedQueryHandler struct {
	projection *projection.Projection
}

type NewEventSourcedQueryHandlerInput struct {
	Log eventsource.Log
	// LegacyNamespace is assigned to uploads recorded before namespaces existed.
	LegacyNamespace string
}

func NewEventSourcedQueryHandler(input NewEventSourcedQueryHandlerInput) *EventSourcedQueryHandler {
	return &EventSourcedQueryHandler{
		projection: projection.New(projection.NewInput{
			Log:             input.Log,
			LegacyNamespace: input.LegacyNamespace,
		}),
	}
}

func (handler *EventSourcedQueryHandler) GetUpload(ctx context.Context, uploadID string) (*upload.Object, error) {
	handler.projection.CatchUp(ctx)
	object, ok := handler.projection.GetUpload(uploadID)
	if !ok {
		return nil, ErrUploadNotFound
	}
	return object, nil
}

func (handler *EventSourcedQueryHandler) GetFileInfo(ctx context.Context, fileID string) (*fileinfo.FileInfo, error) {
	handler.projection.CatchUp(ctx)
	fileInfo, ok := handler.projection.GetFileInfo(fileID)
	if !ok {
		return nil, ErrFileNotFound
	}
	return fileInfo, nil
}
