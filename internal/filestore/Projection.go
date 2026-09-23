package filestore

import (
	"context"

	"github.com/kduong-dev/goutil/eventsource"
	"github.com/kduong-dev/goutil/eventsource/subscription"
	"github.com/kduong-dev/goutil/fatal"
)

// projection folds the event log into upload and file state. Command and
// query handlers each own one so they can catch up independently.
type projection struct {
	log             eventsource.Log
	legacyNamespace string
	cursor          int64
	uploadByID      map[string]*Upload
	fileByID        map[string]*File
}

func newProjection(log eventsource.Log, legacyNamespace string) *projection {
	return &projection{
		log:             log,
		legacyNamespace: legacyNamespace,
		uploadByID:      make(map[string]*Upload),
		fileByID:        make(map[string]*File),
	}
}

func (projection *projection) catchUp(ctx context.Context) {
	var err error
	projection.cursor, err = subscription.CatchUp(ctx, subscription.Input{
		Log:    projection.log,
		Cursor: projection.cursor,
		Apply:  projection.apply,
	})
	fatal.OnError(err)
}

func (projection *projection) apply(ctx context.Context, event *eventsource.Event) error {
	var frame EventFrame
	fatal.UnlessUnmarshal(event.Data, &frame)
	switch frame.Type {
	case EventTypeUploadInitiated:
		projection.applyInitiated(frame.UploadInitiatedEvent)
	case EventTypePartUploaded:
		projection.applyPartUploaded(frame.PartUploadedEvent)
	case EventTypeUploadCompleted:
		projection.applyCompleted(frame.UploadCompletedEvent)
	case EventTypeUploadAborted:
		projection.applyAborted(frame.UploadAbortedEvent)
	}
	return nil
}

func (projection *projection) applyInitiated(event *UploadInitiatedEvent) {
	namespace := event.Namespace
	if namespace == "" {
		// Events written before namespaces existed carried a user id instead.
		namespace = projection.legacyNamespace
	}
	projection.uploadByID[event.UploadID] = &Upload{
		ID:          event.UploadID,
		Namespace:   namespace,
		Key:         event.Key,
		ContentType: event.ContentType,
		Status:      UploadStatusInitiated,
		CreatedAt:   event.CreatedAt,
		UpdatedAt:   event.CreatedAt,
	}
}

func (projection *projection) applyPartUploaded(event *PartUploadedEvent) {
	upload, ok := projection.uploadByID[event.UploadID]
	if !ok {
		return
	}
	part := Part{Number: event.PartNumber, Size: event.Size, Checksum: event.Checksum}
	upload.UpdatedAt = event.UpdatedAt
	for index, existing := range upload.Parts {
		if existing.Number == event.PartNumber {
			upload.Parts[index] = part
			return
		}
	}
	upload.Parts = append(upload.Parts, part)
}

func (projection *projection) applyCompleted(event *UploadCompletedEvent) {
	upload, ok := projection.uploadByID[event.UploadID]
	if !ok {
		return
	}
	upload.Status = UploadStatusCompleted
	upload.UpdatedAt = event.UpdatedAt
	projection.fileByID[event.FileID] = &File{
		ID:          event.FileID,
		Namespace:   upload.Namespace,
		UploadID:    event.UploadID,
		Key:         upload.Key,
		ContentType: upload.ContentType,
		Size:        event.Size,
		Checksum:    event.Checksum,
		CreatedAt:   event.UpdatedAt,
	}
}

func (projection *projection) applyAborted(event *UploadAbortedEvent) {
	upload, ok := projection.uploadByID[event.UploadID]
	if !ok {
		return
	}
	upload.Status = UploadStatusAborted
	upload.UpdatedAt = event.UpdatedAt
}
