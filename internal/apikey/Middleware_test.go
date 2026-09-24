package apikey_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/kduong-dev/storage-service/internal/apikey"
	. "github.com/smartystreets/goconvey/convey"
)

func TestMiddleware(t *testing.T) {
	Convey("Given a middleware with an API key issued to the alpha-service namespace", t, func() {
		middleware := apikey.NewMiddleware(apikey.NewMiddlewareInput{
			NamespaceByKeyHash: map[string]string{apikey.HashAPIKey("secret-key"): "alpha-service"},
		})
		var observedNamespace string
		handler := middleware.Handle(http.HandlerFunc(func(responseWriter http.ResponseWriter, request *http.Request) {
			observedNamespace = apikey.GetNamespace(request.Context())
		}))
		serve := func(authorization string) *httptest.ResponseRecorder {
			request := httptest.NewRequest(http.MethodGet, "/storage/v1/files/file-1", nil)
			if authorization != "" {
				request.Header.Set("Authorization", authorization)
			}
			responseRecorder := httptest.NewRecorder()
			handler.ServeHTTP(responseRecorder, request)
			return responseRecorder
		}
		Convey("When a request presents the issued key", func() {
			responseRecorder := serve("Bearer secret-key")
			Convey("Then the request reaches the handler scoped to that namespace", func() {
				So(responseRecorder.Code, ShouldEqual, http.StatusOK)
				So(observedNamespace, ShouldEqual, "alpha-service")
			})
		})
		Convey("When a request presents an unknown key", func() {
			responseRecorder := serve("Bearer wrong-key")
			Convey("Then it is rejected as unauthorized without reaching the handler", func() {
				So(responseRecorder.Code, ShouldEqual, http.StatusUnauthorized)
				So(observedNamespace, ShouldBeEmpty)
			})
		})
		Convey("When a request presents no key", func() {
			responseRecorder := serve("")
			Convey("Then it is rejected as unauthorized", func() {
				So(responseRecorder.Code, ShouldEqual, http.StatusUnauthorized)
			})
		})
		Convey("When a request uses a non-bearer scheme", func() {
			responseRecorder := serve("Basic secret-key")
			Convey("Then it is rejected as unauthorized", func() {
				So(responseRecorder.Code, ShouldEqual, http.StatusUnauthorized)
			})
		})
	})
}
