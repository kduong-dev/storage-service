package filestore

import (
	"context"

	"github.com/kduong-dev/goutil/eventsource"
	"github.com/kduong-dev/goutil/fatal"
)

var _ CommandHandler = (*EventSourcedCommandHandler)(nil)

type EventSourcedCommandHandler struct {
	projection *projection
}

type NewEventSourcedCommandHandlerInput struct {
	Log eventsource.Log
	// LegacyNamespace is assigned to uploads recorded before namespaces existed.
	LegacyNamespace string
}

func NewEventSourcedCommandHandler(input NewEventSourcedCommandHandlerInput) *EventSourcedCommandHandler {
	return &EventSourcedCommandHandler{projection: newProjection(input.Log, input.LegacyNamespace)}
}

func (handler *EventSourcedCommandHandler) InitialiseUpload(ctx context.Context, upload *Upload) error {
	handler.projection.catchUp(ctx)
	return handler.append(EventFrame{
		EventBase: eventsource.NewEventBase(EventTypeUploadInitiated),
		UploadInitiatedEvent: &UploadInitiatedEvent{
			UploadID:    upload.ID,
			Namespace:   upload.Namespace,
			Key:         upload.Key,
			ContentType: upload.ContentType,
			CreatedAt:   upload.CreatedAt,
		},
	})
}

func (handler *EventSourcedCommandHandler) RecordPart(ctx context.Context, uploadID string, part Part, updatedAt string) error {
	handler.projection.catchUp(ctx)
	if err := handler.assertUploadActive(uploadID); err != nil {
		return err
	}
	return handler.append(EventFrame{
		EventBase: eventsource.NewEventBase(EventTypePartUploaded),
		PartUploadedEvent: &PartUploadedEvent{
			UploadID:   uploadID,
			PartNumber: part.Number,
			Size:       part.Size,
			Checksum:   part.Checksum,
			UpdatedAt:  updatedAt,
		},
	})
}

func (handler *EventSourcedCommandHandler) CompleteUpload(ctx context.Context, input CompleteUploadInput) error {
	handler.projection.catchUp(ctx)
	if err := handler.assertUploadActive(input.UploadID); err != nil {
		return err
	}
	return handler.append(EventFrame{
		EventBase: eventsource.NewEventBase(EventTypeUploadCompleted),
		UploadCompletedEvent: &UploadCompletedEvent{
			UploadID:  input.UploadID,
			FileID:    input.FileID,
			Size:      input.Size,
			Checksum:  input.Checksum,
			UpdatedAt: input.UpdatedAt,
		},
	})
}

func (handler *EventSourcedCommandHandler) AbortUpload(ctx context.Context, uploadID string, updatedAt string) error {
	handler.projection.catchUp(ctx)
	if err := handler.assertUploadActive(uploadID); err != nil {
		return err
	}
	return handler.append(EventFrame{
		EventBase: eventsource.NewEventBase(EventTypeUploadAborted),
		UploadAbortedEvent: &UploadAbortedEvent{
			UploadID:  uploadID,
			UpdatedAt: updatedAt,
		},
	})
}

func (handler *EventSourcedCommandHandler) append(frame EventFrame) error {
	_, err := handler.projection.log.Append(fatal.UnlessMarshal(frame))
	return err
}

func (handler *EventSourcedCommandHandler) assertUploadActive(uploadID string) error {
	upload, ok := handler.projection.uploadByID[uploadID]
	if !ok {
		return ErrUploadNotFound
	}
	if upload.Status != UploadStatusInitiated {
		return ErrUploadNotActive
	}
	return nil
}
