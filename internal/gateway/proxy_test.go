package gateway

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestProxyForwardsToConfiguredUpstream(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/target/graphql" {
			t.Fatalf("unexpected upstream path: %s", r.URL.Path)
		}
		if r.URL.Query().Get("op") != "Account" {
			t.Fatalf("missing op query: %s", r.URL.RawQuery)
		}
		if r.Header.Get("Authorization") != "Bearer test" {
			t.Fatalf("authorization header not forwarded")
		}
		_, _ = w.Write([]byte(`{"data":{"ok":true}}`))
	}))
	defer upstream.Close()

	target, err := url.Parse(upstream.URL + "/target/graphql")
	if err != nil {
		t.Fatal(err)
	}

	cfg := Config{
		Addr:           ":0",
		AllowedOrigins: []string{"*"},
		RequestTimeout: time.Second,
		Routes: []Route{{
			Name:       "core",
			Path:       "/api/meme/graphql",
			Upstream:   target,
			UpstreamID: "XBIT_UPSTREAM_CORE_GRAPHQL_URL",
		}},
	}

	router := NewRouter(cfg, slog.Default())
	req := httptest.NewRequest(http.MethodPost, "/api/meme/graphql?op=Account", strings.NewReader(`{"query":"query Account { ok }"}`))
	req.Header.Set("Authorization", "Bearer test")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", rec.Code, rec.Body.String())
	}
	body, _ := io.ReadAll(rec.Body)
	if !strings.Contains(string(body), `"ok":true`) {
		t.Fatalf("unexpected body: %s", string(body))
	}
}

func TestProxyReturnsGraphQLErrorWhenUpstreamMissing(t *testing.T) {
	cfg := Config{
		Addr:           ":0",
		AllowedOrigins: []string{"*"},
		RequestTimeout: time.Second,
		Routes: []Route{{
			Name:       "core",
			Path:       "/api/meme/graphql",
			UpstreamID: "XBIT_UPSTREAM_CORE_GRAPHQL_URL",
		}},
	}

	router := NewRouter(cfg, slog.Default())
	req := httptest.NewRequest(http.MethodPost, "/api/meme/graphql", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadGateway {
		t.Fatalf("status = %d body = %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), CodeUpstreamNotConfiguredCompat) {
		t.Fatalf("expected upstream code in body: %s", rec.Body.String())
	}
}

const CodeUpstreamNotConfiguredCompat = "ERR_UPSTREAM_NOT_CONFIGURED"
