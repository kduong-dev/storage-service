package filestore

import (
	"context"

	"github.com/kduong/trading-backend/internal/contextx"
	"github.com/kduong/trading-backend/internal/eventsource"
	"github.com/kduong/trading-backend/internal/eventsource/subscription"
	"github.com/kduong/trading-backend/internal/fatal"
)

var _ QueryHandler = (*EventSourcedQueryHandler)(nil)

type EventSourcedQueryHandler struct {
	log        eventsource.Log
	cursor     int64
	uploadByID map[string]*Upload
	fileByID   map[string]*File
}

type NewEventSourcedQueryHandlerInput struct {
	Log eventsource.Log
}

func NewEventSourcedQueryHandler(input NewEventSourcedQueryHandlerInput) *EventSourcedQueryHandler {
	return &EventSourcedQueryHandler{
		log:        input.Log,
		uploadByID: make(map[string]*Upload),
		fileByID:   make(map[string]*File),
	}
}

func (handler *EventSourcedQueryHandler) GetUpload(ctx context.Context, uploadID string) (*Upload, error) {
	handler.catchUp(ctx)
	upload, ok := handler.uploadByID[uploadID]
	if !ok {
		return nil, ErrUploadNotFound
	}
	userID := contextx.GetUserID(ctx)
	if upload.UserID != userID {
		return nil, ErrUploadForbidden
	}
	return upload, nil
}

func (handler *EventSourcedQueryHandler) GetFile(ctx context.Context, fileID string) (*File, error) {
	handler.catchUp(ctx)
	file, ok := handler.fileByID[fileID]
	if !ok {
		return nil, ErrFileNotFound
	}
	userID := contextx.GetUserID(ctx)
	if file.UserID != userID {
		return nil, ErrFileForbidden
	}
	return file, nil
}

func (handler *EventSourcedQueryHandler) catchUp(ctx context.Context) {
	var err error
	handler.cursor, err = subscription.CatchUp(ctx, subscription.Input{
		Log:    handler.log,
		Cursor: handler.cursor,
		Apply:  handler.apply,
	})
	fatal.OnError(err)
}

func (handler *EventSourcedQueryHandler) apply(ctx context.Context, event *eventsource.Event) error {
	var frame EventFrame
	fatal.UnlessUnmarshal(event.Data, &frame)
	switch frame.Type {
	case EventTypeUploadInitiated:
		return handler.applyInitiated(frame.UploadInitiatedEvent)
	case EventTypePartUploaded:
		return handler.applyPartUploaded(frame.PartUploadedEvent)
	case EventTypeUploadCompleted:
		return handler.applyCompleted(frame.UploadCompletedEvent)
	case EventTypeUploadAborted:
		return handler.applyAborted(frame.UploadAbortedEvent)
	}
	return nil
}

func (handler *EventSourcedQueryHandler) applyInitiated(e *UploadInitiatedEvent) error {
	handler.uploadByID[e.UploadID] = &Upload{
		ID:          e.UploadID,
		UserID:      e.UserID,
		Filename:    e.Filename,
		ContentType: e.ContentType,
		Status:      UploadStatusInitiated,
		CreatedAt:   e.CreatedAt,
		UpdatedAt:   e.CreatedAt,
	}
	return nil
}

func (handler *EventSourcedQueryHandler) applyPartUploaded(e *PartUploadedEvent) error {
	upload, ok := handler.uploadByID[e.UploadID]
	if !ok {
		return nil
	}
	for i, part := range upload.Parts {
		if part.Number == e.PartNumber {
			upload.Parts[i] = Part{Number: e.PartNumber, Size: e.Size, Checksum: e.Checksum}
			upload.UpdatedAt = e.UpdatedAt
			return nil
		}
	}
	upload.Parts = append(upload.Parts, Part{Number: e.PartNumber, Size: e.Size, Checksum: e.Checksum})
	upload.UpdatedAt = e.UpdatedAt
	return nil
}

func (handler *EventSourcedQueryHandler) applyCompleted(e *UploadCompletedEvent) error {
	upload, ok := handler.uploadByID[e.UploadID]
	if !ok {
		return nil
	}
	upload.Status = UploadStatusCompleted
	upload.UpdatedAt = e.UpdatedAt
	handler.fileByID[e.FileID] = &File{
		ID:          e.FileID,
		UserID:      upload.UserID,
		UploadID:    e.UploadID,
		Filename:    upload.Filename,
		ContentType: upload.ContentType,
		Size:        e.Size,
		Checksum:    e.Checksum,
		CreatedAt:   e.UpdatedAt,
	}
	return nil
}

func (handler *EventSourcedQueryHandler) applyAborted(event *UploadAbortedEvent) error {
	upload, ok := handler.uploadByID[event.UploadID]
	if !ok {
		return nil
	}
	upload.Status = UploadStatusAborted
	upload.UpdatedAt = event.UpdatedAt
	return nil
}
