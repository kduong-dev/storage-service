package filestore

import (
	"context"

	"github.com/kduong/trading-backend/internal/eventsource"
	"github.com/kduong/trading-backend/internal/eventsource/subscription"
	"github.com/kduong/trading-backend/internal/fatal"
)

var _ CommandHandler = (*EventSourcedCommandHandler)(nil)

type EventSourcedCommandHandler struct {
	log        eventsource.Log
	cursor     int64
	uploadByID map[string]*Upload
	fileByID   map[string]*File
}

type NewEventSourcedCommandHandlerInput struct {
	Log eventsource.Log
}

func NewEventSourcedCommandHandler(input NewEventSourcedCommandHandlerInput) *EventSourcedCommandHandler {
	return &EventSourcedCommandHandler{
		log:        input.Log,
		uploadByID: make(map[string]*Upload),
		fileByID:   make(map[string]*File),
	}
}

func (handler *EventSourcedCommandHandler) InitialiseUpload(ctx context.Context, upload *Upload) error {
	handler.catchUp(ctx)
	payload := fatal.UnlessMarshal(EventFrame{
		EventBase: eventsource.NewEventBase(EventTypeUploadInitiated),
		UploadInitiatedEvent: &UploadInitiatedEvent{
			UploadID:    upload.ID,
			UserID:      upload.UserID,
			Filename:    upload.Filename,
			ContentType: upload.ContentType,
			CreatedAt:   upload.CreatedAt,
		},
	})
	_, err := handler.log.Append(payload)
	return err
}

func (handler *EventSourcedCommandHandler) RecordPart(ctx context.Context, uploadID string, part Part, updatedAt string) error {
	handler.catchUp(ctx)
	if err := handler.assertUploadActive(uploadID); err != nil {
		return err
	}
	payload := fatal.UnlessMarshal(EventFrame{
		EventBase: eventsource.NewEventBase(EventTypePartUploaded),
		PartUploadedEvent: &PartUploadedEvent{
			UploadID:   uploadID,
			PartNumber: part.PartNumber,
			Size:       part.Size,
			Checksum:   part.Checksum,
			UpdatedAt:  updatedAt,
		},
	})
	_, err := handler.log.Append(payload)
	return err
}

func (handler *EventSourcedCommandHandler) CompleteUpload(ctx context.Context, uploadID string, fileID string, size int64, checksum string, updatedAt string) error {
	handler.catchUp(ctx)
	if err := handler.assertUploadActive(uploadID); err != nil {
		return err
	}
	payload := fatal.UnlessMarshal(EventFrame{
		EventBase: eventsource.NewEventBase(EventTypeUploadCompleted),
		UploadCompletedEvent: &UploadCompletedEvent{
			UploadID:  uploadID,
			FileID:    fileID,
			Size:      size,
			Checksum:  checksum,
			UpdatedAt: updatedAt,
		},
	})
	_, err := handler.log.Append(payload)
	return err
}

func (handler *EventSourcedCommandHandler) AbortUpload(ctx context.Context, uploadID string, updatedAt string) error {
	handler.catchUp(ctx)
	if err := handler.assertUploadActive(uploadID); err != nil {
		return err
	}
	payload := fatal.UnlessMarshal(EventFrame{
		EventBase: eventsource.NewEventBase(EventTypeUploadAborted),
		UploadAbortedEvent: &UploadAbortedEvent{
			UploadID:  uploadID,
			UpdatedAt: updatedAt,
		},
	})
	_, err := handler.log.Append(payload)
	return err
}

func (handler *EventSourcedCommandHandler) assertUploadActive(uploadID string) error {
	upload, ok := handler.uploadByID[uploadID]
	if !ok {
		return ErrUploadNotFound
	}
	if upload.Status != UploadStatusInitiated {
		return ErrUploadNotActive
	}
	return nil
}

func (handler *EventSourcedCommandHandler) catchUp(ctx context.Context) {
	var err error
	handler.cursor, err = subscription.CatchUp(ctx, subscription.Input{
		Log:    handler.log,
		Cursor: handler.cursor,
		Apply:  handler.apply,
	})
	fatal.OnError(err)
}

func (handler *EventSourcedCommandHandler) apply(ctx context.Context, event *eventsource.Event) error {
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

func (handler *EventSourcedCommandHandler) applyInitiated(event *UploadInitiatedEvent) error {
	handler.uploadByID[event.UploadID] = &Upload{
		ID:          event.UploadID,
		UserID:      event.UserID,
		Filename:    event.Filename,
		ContentType: event.ContentType,
		Status:      UploadStatusInitiated,
		CreatedAt:   event.CreatedAt,
		UpdatedAt:   event.CreatedAt,
	}
	return nil
}

func (handler *EventSourcedCommandHandler) applyPartUploaded(event *PartUploadedEvent) error {
	upload, ok := handler.uploadByID[event.UploadID]
	if !ok {
		return nil
	}
	// Replace if the same part number was re-uploaded.
	for i, p := range upload.Parts {
		if p.PartNumber == event.PartNumber {
			upload.Parts[i] = Part{PartNumber: event.PartNumber, Size: event.Size, Checksum: event.Checksum}
			upload.UpdatedAt = event.UpdatedAt
			return nil
		}
	}
	upload.Parts = append(upload.Parts, Part{PartNumber: event.PartNumber, Size: event.Size, Checksum: event.Checksum})
	upload.UpdatedAt = event.UpdatedAt
	return nil
}

func (handler *EventSourcedCommandHandler) applyCompleted(event *UploadCompletedEvent) error {
	upload, ok := handler.uploadByID[event.UploadID]
	if !ok {
		return nil
	}
	upload.Status = UploadStatusCompleted
	upload.UpdatedAt = event.UpdatedAt
	// Materialise the File record.
	handler.fileByID[event.FileID] = &File{
		ID:          event.FileID,
		UserID:      upload.UserID,
		UploadID:    event.UploadID,
		Filename:    upload.Filename,
		ContentType: upload.ContentType,
		Size:        event.Size,
		Checksum:    event.Checksum,
		CreatedAt:   event.UpdatedAt,
	}
	return nil
}

func (handler *EventSourcedCommandHandler) applyAborted(event *UploadAbortedEvent) error {
	upload, ok := handler.uploadByID[event.UploadID]
	if !ok {
		return nil
	}
	upload.Status = UploadStatusAborted
	upload.UpdatedAt = event.UpdatedAt
	return nil
}
