package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestBearerMiddlewareInjectsClaims(t *testing.T) {
	manager, err := NewTokenManager(TokenManagerConfig{
		SigningKey: "secret",
		AccessTTL:  time.Minute,
		RefreshTTL: time.Hour,
	})
	if err != nil {
		t.Fatal(err)
	}
	pair, err := manager.IssuePair("user-1", "session-1", "", "Web", nil)
	if err != nil {
		t.Fatal(err)
	}

	handler := BearerMiddleware(manager, nil)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims, ok := ClaimsFromContext(r.Context())
		if !ok || claims.UserID != "user-1" {
			t.Fatalf("claims missing: %+v", claims)
		}
		w.WriteHeader(http.StatusNoContent)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+pair.AccessToken)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d", rec.Code)
	}
}

func TestBearerMiddlewareRejectsMissingToken(t *testing.T) {
	manager, err := NewTokenManager(TokenManagerConfig{
		SigningKey: "secret",
		AccessTTL:  time.Minute,
		RefreshTTL: time.Hour,
	})
	if err != nil {
		t.Fatal(err)
	}

	handler := BearerMiddleware(manager, nil)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("handler should not run")
	}))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d", rec.Code)
	}
}
