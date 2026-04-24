package httpapi

import (
	"encoding/json"
	"net/http"
	"sort"
	"time"

	"github.com/ansel1/merry"
	"github.com/gorilla/mux"
	"github.com/kduong/trading-backend/internal/authz"
	"github.com/kduong/trading-backend/internal/httpx"
	uuid "github.com/satori/go.uuid"
)

func (handler *Handler) CompleteUpload(responseWriter http.ResponseWriter, request *http.Request) {
	var err error
	defer func() {
		if err != nil {
			httpx.SendErrorResponse(responseWriter, err)
		}
	}()
	ctx := request.Context()
	if err = authz.RequireScope(ctx, authz.ScopeFilesWrite); err != nil {
		return
	}
	uploadID := mux.Vars(request)["upload_id"]

	// Verify ownership and fetch recorded parts.
	upload, err := handler.queryHandler.GetUpload(ctx, uploadID)
	if err != nil {
		err = merrifyError[err]
		return
	}
	if len(upload.Parts) == 0 {
		err = merry.New("no parts have been uploaded").WithHTTPCode(http.StatusBadRequest)
		return
	}

	partNumbers := make([]int, len(upload.Parts))
	for i, part := range upload.Parts {
		partNumbers[i] = part.Number
	}
	sort.Ints(partNumbers)

	fileID := uuid.NewV4().String()
	size, checksum, backendErr := handler.backend.Assemble(uploadID, fileID, partNumbers)
	if backendErr != nil {
		err = merry.Wrap(backendErr).WithHTTPCode(http.StatusInternalServerError)
		return
	}

	now := time.Now().UTC().Format(time.RFC3339)
	if err = handler.commandHandler.CompleteUpload(ctx, uploadID, fileID, size, checksum, now); err != nil {
		err = merrifyError[err]
		return
	}

	// Clean up temporary part data; non-fatal on error.
	handler.backend.DeleteParts(uploadID)

	// Re-read the finalised file record from the query handler.
	file, err := handler.queryHandler.GetFile(ctx, fileID)
	if err != nil {
		err = merrifyError[err]
		return
	}

	responseWriter.Header().Set("Content-Type", "application/json")
	responseWriter.WriteHeader(http.StatusCreated)
	json.NewEncoder(responseWriter).Encode(file)
}
