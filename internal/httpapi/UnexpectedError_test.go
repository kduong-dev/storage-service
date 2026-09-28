package httpapi_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/kduong-dev/storage-service/internal/apikey"
	"github.com/kduong-dev/storage-service/internal/file"
	"github.com/kduong-dev/storage-service/internal/httpapi"
	"github.com/kduong-dev/storage-service/pkg/storageservice"
	. "github.com/smartystreets/goconvey/convey"
)

// failingFileObjectStore fails every Get with an error no sentinel matches,
// as a file store whose log is unreachable does.
type failingFileObjectStore struct {
	file.ObjectStore
}

func (failingFileObjectStore) Get(ctx context.Context, fileID string) (*storageservice.FileObject, error) {
	return nil, errors.New("log unavailable")
}

func TestUnexpectedError(t *testing.T) {
	Convey("Given a storage service whose file store is failing", t, func() {
		handler := httpapi.NewHandler(httpapi.NewHandlerInput{
			APIKeyMiddleware: apikey.NewMiddleware(apikey.NewMiddlewareInput{
				NamespaceByKeyHash: map[string]string{apikey.HashAPIKey("alpha-key"): "alpha-service"},
			}),
			FileObjectStore: failingFileObjectStore{},
		})
		Convey("When getting a file's metadata", func() {
			request := httptest.NewRequest(http.MethodGet, "/storage/v1/files/file-1/metadata", nil)
			request.Header.Set("Authorization", "Bearer alpha-key")
			recorder := httptest.NewRecorder()
			handler.ServeHTTP(recorder, request)
			Convey("Then it responds internal server error rather than exiting", func() {
				So(recorder.Code, ShouldEqual, http.StatusInternalServerError)
			})
		})
	})
}
