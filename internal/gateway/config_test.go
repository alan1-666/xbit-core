package gateway

import (
	"strings"
	"testing"
)

func TestDefaultRouteDefinitionsContainFrontendEndpoints(t *testing.T) {
	defs := DefaultRouteDefinitions()
	if len(defs) < 15 {
		t.Fatalf("expected frontend endpoint coverage, got %d routes", len(defs))
	}

	paths := map[string]bool{}
	for _, def := range defs {
		if def.Name == "" || def.Path == "" || def.Env == "" {
			t.Fatalf("route definition has empty field: %+v", def)
		}
		if !strings.HasPrefix(def.Path, "/api/") {
			t.Fatalf("route path should be an API path: %+v", def)
		}
		paths[def.Path] = true
	}

	for _, path := range []string{
		"/api/meme/graphql",
		"/api/meme2/meme-gql",
		"/api/trading/trading-gql",
		"/api/user/user-gql",
		"/api/xp-api/predict-gql",
		"/api/xp-user-service",
	} {
		if !paths[path] {
			t.Fatalf("missing frontend path %s", path)
		}
	}
}

func TestLoadConfigFromEnv(t *testing.T) {
	t.Setenv("GATEWAY_ADDR", ":9999")
	t.Setenv("GATEWAY_REQUEST_TIMEOUT", "7s")
	t.Setenv("GATEWAY_ALLOWED_ORIGINS", "http://localhost:4200,http://localhost:5173")
	t.Setenv("XBIT_UPSTREAM_CORE_GRAPHQL_URL", "https://example.com/api/meme/graphql")

	cfg, err := LoadConfigFromEnv()
	if err != nil {
		t.Fatalf("LoadConfigFromEnv returned error: %v", err)
	}
	if cfg.Addr != ":9999" {
		t.Fatalf("addr = %s", cfg.Addr)
	}
	if cfg.RequestTimeout.String() != "7s" {
		t.Fatalf("timeout = %s", cfg.RequestTimeout)
	}
	if len(cfg.AllowedOrigins) != 2 {
		t.Fatalf("allowed origins = %#v", cfg.AllowedOrigins)
	}
	if cfg.Routes[0].Upstream == nil || cfg.Routes[0].Upstream.Host != "example.com" {
		t.Fatalf("core upstream not loaded: %+v", cfg.Routes[0].Upstream)
	}
}
