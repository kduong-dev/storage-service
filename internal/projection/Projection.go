package projection

import (
	"context"

	"github.com/kduong-dev/goutil/eventsource"
	"github.com/kduong-dev/goutil/eventsource/subscription"
	"github.com/kduong-dev/goutil/fatal"
	"github.com/kduong-dev/storage-service/internal/fileinfo"
	"github.com/kduong-dev/storage-service/internal/upload"
)

// Projection folds the event log into upload and file info state. Command and
// query handlers each own one so they can catch up independently.
type Projection struct {
	log              eventsource.Log
	legacyNamespace  string
	cursor           int64
	uploadByID       map[string]*upload.Object
	fileInfoByFileID map[string]*fileinfo.FileInfo
}

type NewInput struct {
	Log eventsource.Log
	// LegacyNamespace is assigned to uploads recorded before namespaces existed.
	LegacyNamespace string
}

func New(input NewInput) *Projection {
	return &Projection{
		log:              input.Log,
		legacyNamespace:  input.LegacyNamespace,
		uploadByID:       make(map[string]*upload.Object),
		fileInfoByFileID: make(map[string]*fileinfo.FileInfo),
	}
}

func (projection *Projection) CatchUp(ctx context.Context) {
	var err error
	projection.cursor, err = subscription.CatchUp(ctx, subscription.Input{
		Log:    projection.log,
		Cursor: projection.cursor,
		Apply:  projection.apply,
	})
	fatal.OnError(err)
}

func (projection *Projection) GetUpload(uploadID string) (*upload.Object, bool) {
	object, ok := projection.uploadByID[uploadID]
	return object, ok
}

func (projection *Projection) GetFileInfo(fileID string) (*fileinfo.FileInfo, bool) {
	fileInfo, ok := projection.fileInfoByFileID[fileID]
	return fileInfo, ok
}

func (projection *Projection) apply(ctx context.Context, event *eventsource.Event) error {
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

func (projection *Projection) applyInitiated(event *UploadInitiatedEvent) {
	namespace := event.Namespace
	if namespace == "" {
		// Events written before namespaces existed carried a user id instead.
		namespace = projection.legacyNamespace
	}
	projection.uploadByID[event.UploadID] = &upload.Object{
		ID:          event.UploadID,
		Namespace:   namespace,
		Key:         event.Key,
		ContentType: event.ContentType,
		Status:      upload.StatusInitiated,
		CreatedAt:   event.CreatedAt,
		UpdatedAt:   event.CreatedAt,
	}
}

func (projection *Projection) applyPartUploaded(event *PartUploadedEvent) {
	object, ok := projection.uploadByID[event.UploadID]
	if !ok {
		return
	}
	part := upload.Part{Number: event.PartNumber, Size: event.Size, Checksum: event.Checksum}
	object.UpdatedAt = event.UpdatedAt
	for index, existing := range object.Parts {
		if existing.Number == event.PartNumber {
			object.Parts[index] = part
			return
		}
	}
	object.Parts = append(object.Parts, part)
}

func (projection *Projection) applyCompleted(event *UploadCompletedEvent) {
	object, ok := projection.uploadByID[event.UploadID]
	if !ok {
		return
	}
	object.Status = upload.StatusCompleted
	object.UpdatedAt = event.UpdatedAt
	projection.fileInfoByFileID[event.FileID] = &fileinfo.FileInfo{
		ID:          event.FileID,
		Namespace:   object.Namespace,
		UploadID:    event.UploadID,
		Key:         object.Key,
		ContentType: object.ContentType,
		Size:        event.Size,
		Checksum:    event.Checksum,
		CreatedAt:   event.UpdatedAt,
	}
}

func (projection *Projection) applyAborted(event *UploadAbortedEvent) {
	object, ok := projection.uploadByID[event.UploadID]
	if !ok {
		return
	}
	object.Status = upload.StatusAborted
	object.UpdatedAt = event.UpdatedAt
}
