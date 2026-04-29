package errors

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/xbit/xbit-backend/pkg/requestid"
)

func TestWriteGraphQLErrorIncludesCodeAndTraceID(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/graphql", nil)
	req = req.WithContext(requestid.WithContext(req.Context(), "trace-test"))
	rec := httptest.NewRecorder()

	WriteGraphQLError(rec, req, http.StatusUnauthorized, CodeUnauthenticated, "login required", nil)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d", rec.Code)
	}
	body := rec.Body.String()
	for _, want := range []string{CodeUnauthenticated, "trace-test", "login required"} {
		if !strings.Contains(body, want) {
			t.Fatalf("body %q does not contain %q", body, want)
		}
	}
}
