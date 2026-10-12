// External test package: the test sees httpapi from the outside,
// like any other package, and can use only exported names.
package httpapi_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/IvanShchadov/wallet/internal/httpapi"
)

func TestRouter(t *testing.T) {
	tests := []struct {
		name       string
		method     string
		path       string
		wantStatus int
		wantType   string // empty = do not check Content-Type
		wantBody   string // empty = do not check body
	}{
		{"healthz ok", http.MethodGet, "/healthz", http.StatusOK, "text/plain; charset=utf-8", "ok"},
		{"healthz wrong method", http.MethodPost, "/healthz", http.StatusMethodNotAllowed, "", ""},
		{"unknown route", http.MethodGet, "/nope", http.StatusNotFound, "", ""},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h := httpapi.NewRouter()

			// No network: httptest builds the request in memory,
			// and the recorder captures everything the handler writes.
			req := httptest.NewRequest(tc.method, tc.path, nil)
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, req)

			// Result() is the response as the client sees it. Its headers are
			// a snapshot taken when the response was sent, so a header set
			// after WriteHeader is missing here, just like on the wire.
			res := rec.Result()
			defer res.Body.Close()

			body, err := io.ReadAll(res.Body)
			if err != nil {
				t.Fatalf("read body: %v", err)
			}

			if res.StatusCode != tc.wantStatus {
				t.Fatalf("%s %s: status = %d, want %d", tc.method, tc.path, res.StatusCode, tc.wantStatus)
			}
			if got := res.Header.Get("Content-Type"); tc.wantType != "" && got != tc.wantType {
				t.Fatalf("%s %s: Content-Type = %q, want %q", tc.method, tc.path, got, tc.wantType)
			}
			if tc.wantBody != "" && string(body) != tc.wantBody {
				t.Fatalf("%s %s: body = %q, want %q", tc.method, tc.path, body, tc.wantBody)
			}
		})
	}
}
