package fileinfostore

import (
	"context"

	"github.com/kduong-dev/goutil/eventsource"
	"github.com/kduong-dev/goutil/fatal"
	"github.com/kduong-dev/storage-service/internal/projection"
	"github.com/kduong-dev/storage-service/internal/upload"
)

var _ CommandHandler = (*EventSourcedCommandHandler)(nil)

type EventSourcedCommandHandler struct {
	log        eventsource.Log
	projection *projection.Projection
}

type NewEventSourcedCommandHandlerInput struct {
	Log eventsource.Log
	// LegacyNamespace is assigned to uploads recorded before namespaces existed.
	LegacyNamespace string
}

func NewEventSourcedCommandHandler(input NewEventSourcedCommandHandlerInput) *EventSourcedCommandHandler {
	return &EventSourcedCommandHandler{
		log: input.Log,
		projection: projection.New(projection.NewInput{
			Log:             input.Log,
			LegacyNamespace: input.LegacyNamespace,
		}),
	}
}

func (handler *EventSourcedCommandHandler) InitialiseUpload(ctx context.Context, object *upload.Object) error {
	handler.projection.CatchUp(ctx)
	return handler.append(projection.EventFrame{
		EventBase: eventsource.NewEventBase(projection.EventTypeUploadInitiated),
		UploadInitiatedEvent: &projection.UploadInitiatedEvent{
			UploadID:    object.ID,
			Namespace:   object.Namespace,
			Key:         object.Key,
			ContentType: object.ContentType,
			CreatedAt:   object.CreatedAt,
		},
	})
}

func (handler *EventSourcedCommandHandler) RecordPart(ctx context.Context, uploadID string, part upload.Part, updatedAt string) error {
	handler.projection.CatchUp(ctx)
	if err := handler.assertUploadActive(uploadID); err != nil {
		return err
	}
	return handler.append(projection.EventFrame{
		EventBase: eventsource.NewEventBase(projection.EventTypePartUploaded),
		PartUploadedEvent: &projection.PartUploadedEvent{
			UploadID:   uploadID,
			PartNumber: part.Number,
			Size:       part.Size,
			Checksum:   part.Checksum,
			UpdatedAt:  updatedAt,
		},
	})
}

func (handler *EventSourcedCommandHandler) CompleteUpload(ctx context.Context, input CompleteUploadInput) error {
	handler.projection.CatchUp(ctx)
	if err := handler.assertUploadActive(input.UploadID); err != nil {
		return err
	}
	return handler.append(projection.EventFrame{
		EventBase: eventsource.NewEventBase(projection.EventTypeUploadCompleted),
		UploadCompletedEvent: &projection.UploadCompletedEvent{
			UploadID:  input.UploadID,
			FileID:    input.FileID,
			Size:      input.Size,
			Checksum:  input.Checksum,
			UpdatedAt: input.UpdatedAt,
		},
	})
}

func (handler *EventSourcedCommandHandler) AbortUpload(ctx context.Context, uploadID string, updatedAt string) error {
	handler.projection.CatchUp(ctx)
	if err := handler.assertUploadActive(uploadID); err != nil {
		return err
	}
	return handler.append(projection.EventFrame{
		EventBase: eventsource.NewEventBase(projection.EventTypeUploadAborted),
		UploadAbortedEvent: &projection.UploadAbortedEvent{
			UploadID:  uploadID,
			UpdatedAt: updatedAt,
		},
	})
}

func (handler *EventSourcedCommandHandler) append(frame projection.EventFrame) error {
	_, err := handler.log.Append(fatal.UnlessMarshal(frame))
	return err
}

func (handler *EventSourcedCommandHandler) assertUploadActive(uploadID string) error {
	object, ok := handler.projection.GetUpload(uploadID)
	if !ok {
		return ErrUploadNotFound
	}
	if object.Status != upload.StatusInitiated {
		return ErrUploadNotActive
	}
	return nil
}
