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

func (h *EventSourcedQueryHandler) GetUpload(ctx context.Context, uploadID string) (*Upload, error) {
	h.catchUp(ctx)
	upload, ok := h.uploadByID[uploadID]
	if !ok {
		return nil, ErrUploadNotFound
	}
	userID := contextx.GetUserID(ctx)
	if upload.UserID != userID {
		return nil, ErrUploadForbidden
	}
	return upload, nil
}

func (h *EventSourcedQueryHandler) GetFile(ctx context.Context, fileID string) (*File, error) {
	h.catchUp(ctx)
	file, ok := h.fileByID[fileID]
	if !ok {
		return nil, ErrFileNotFound
	}
	userID := contextx.GetUserID(ctx)
	if file.UserID != userID {
		return nil, ErrFileForbidden
	}
	return file, nil
}

func (h *EventSourcedQueryHandler) catchUp(ctx context.Context) {
	var err error
	h.cursor, err = subscription.CatchUp(ctx, subscription.Input{
		Log:    h.log,
		Cursor: h.cursor,
		Apply:  h.apply,
	})
	fatal.OnError(err)
}

func (h *EventSourcedQueryHandler) apply(ctx context.Context, event *eventsource.Event) error {
	var frame EventFrame
	fatal.UnlessUnmarshal(event.Data, &frame)
	switch frame.Type {
	case EventTypeUploadInitiated:
		return h.applyInitiated(frame.UploadInitiatedEvent)
	case EventTypePartUploaded:
		return h.applyPartUploaded(frame.PartUploadedEvent)
	case EventTypeUploadCompleted:
		return h.applyCompleted(frame.UploadCompletedEvent)
	case EventTypeUploadAborted:
		return h.applyAborted(frame.UploadAbortedEvent)
	}
	return nil
}

func (h *EventSourcedQueryHandler) applyInitiated(e *UploadInitiatedEvent) error {
	h.uploadByID[e.UploadID] = &Upload{
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

func (h *EventSourcedQueryHandler) applyPartUploaded(e *PartUploadedEvent) error {
	upload, ok := h.uploadByID[e.UploadID]
	if !ok {
		return nil
	}
	for i, p := range upload.Parts {
		if p.PartNumber == e.PartNumber {
			upload.Parts[i] = Part{PartNumber: e.PartNumber, Size: e.Size, Checksum: e.Checksum}
			upload.UpdatedAt = e.UpdatedAt
			return nil
		}
	}
	upload.Parts = append(upload.Parts, Part{PartNumber: e.PartNumber, Size: e.Size, Checksum: e.Checksum})
	upload.UpdatedAt = e.UpdatedAt
	return nil
}

func (h *EventSourcedQueryHandler) applyCompleted(e *UploadCompletedEvent) error {
	upload, ok := h.uploadByID[e.UploadID]
	if !ok {
		return nil
	}
	upload.Status = UploadStatusCompleted
	upload.UpdatedAt = e.UpdatedAt
	h.fileByID[e.FileID] = &File{
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

func (h *EventSourcedQueryHandler) applyAborted(e *UploadAbortedEvent) error {
	upload, ok := h.uploadByID[e.UploadID]
	if !ok {
		return nil
	}
	upload.Status = UploadStatusAborted
	upload.UpdatedAt = e.UpdatedAt
	return nil
}
