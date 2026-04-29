package gateway

import (
	"fmt"
	"net/url"
	"os"
	"strings"
	"time"
)

type RouteDefinition struct {
	Name string
	Path string
	Env  string
}

type Route struct {
	Name       string
	Path       string
	Upstream   *url.URL
	UpstreamID string
}

type Config struct {
	Addr            string
	AllowedOrigins  []string
	RequestTimeout  time.Duration
	Routes          []Route
	RouteDefinition []RouteDefinition
}

func DefaultRouteDefinitions() []RouteDefinition {
	return []RouteDefinition{
		{Name: "core", Path: "/api/meme/graphql", Env: "XBIT_UPSTREAM_CORE_GRAPHQL_URL"},
		{Name: "meme2", Path: "/api/meme2/meme-gql", Env: "XBIT_UPSTREAM_MEME2_GRAPHQL_URL"},
		{Name: "agent", Path: "/api/dex-agent/graphql", Env: "XBIT_UPSTREAM_AGENT_GRAPHQL_URL"},
		{Name: "trading", Path: "/api/trading/trading-gql", Env: "XBIT_UPSTREAM_TRADING_GRAPHQL_URL"},
		{Name: "user", Path: "/api/user/user-gql", Env: "XBIT_UPSTREAM_USER_GRAPHQL_URL"},
		{Name: "notification", Path: "/api/notification/notification-gql", Env: "XBIT_UPSTREAM_NOTIFICATION_GRAPHQL_URL"},
		{Name: "wallet", Path: "/api/wallet/query", Env: "XBIT_UPSTREAM_WALLET_GRAPHQL_URL"},
		{Name: "affiliate", Path: "/api/affiliate/query", Env: "XBIT_UPSTREAM_AFFILIATE_GRAPHQL_URL"},
		{Name: "prediction", Path: "/api/xp-api/predict-gql", Env: "XBIT_UPSTREAM_PREDICTION_GRAPHQL_URL"},
		{Name: "xp-user", Path: "/api/xp-user-service", Env: "XBIT_UPSTREAM_XP_USER_GRAPHQL_URL"},
		{Name: "dex-hypertrader", Path: "/api/dex-hypertrader/graphql", Env: "XBIT_UPSTREAM_DEX_HYPERTRADER_GRAPHQL_URL"},
		{Name: "symbol-dex", Path: "/api/graphql-dex", Env: "XBIT_UPSTREAM_SYMBOL_DEX_GRAPHQL_URL"},
		{Name: "admin", Path: "/api/admin/query", Env: "XBIT_UPSTREAM_ADMIN_GRAPHQL_URL"},
		{Name: "loyalty", Path: "/api/loyalty/query", Env: "XBIT_UPSTREAM_LOYALTY_GRAPHQL_URL"},
		{Name: "redpacket", Path: "/api/campaign/red-packet-gql", Env: "XBIT_UPSTREAM_REDPACKET_GRAPHQL_URL"},
	}
}

func LoadConfigFromEnv() (Config, error) {
	addr := envOr("GATEWAY_ADDR", ":8080")
	timeout, err := time.ParseDuration(envOr("GATEWAY_REQUEST_TIMEOUT", "30s"))
	if err != nil {
		return Config{}, fmt.Errorf("parse GATEWAY_REQUEST_TIMEOUT: %w", err)
	}

	defs := DefaultRouteDefinitions()
	routes := make([]Route, 0, len(defs))
	for _, def := range defs {
		raw := strings.TrimSpace(os.Getenv(def.Env))
		route := Route{
			Name:       def.Name,
			Path:       strings.TrimRight(def.Path, "/"),
			UpstreamID: def.Env,
		}
		if raw != "" {
			parsed, err := url.Parse(raw)
			if err != nil {
				return Config{}, fmt.Errorf("parse %s: %w", def.Env, err)
			}
			if parsed.Scheme == "" || parsed.Host == "" {
				return Config{}, fmt.Errorf("%s must be an absolute URL", def.Env)
			}
			route.Upstream = parsed
		}
		routes = append(routes, route)
	}

	return Config{
		Addr:            addr,
		AllowedOrigins:  splitCSV(envOr("GATEWAY_ALLOWED_ORIGINS", "*")),
		RequestTimeout:  timeout,
		Routes:          routes,
		RouteDefinition: defs,
	}, nil
}

func envOr(key string, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}

func splitCSV(value string) []string {
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	if len(out) == 0 {
		return []string{"*"}
	}
	return out
}
