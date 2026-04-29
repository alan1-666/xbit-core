package gateway

import (
	"log/slog"
	"net/http"
	"net/http/httputil"
	"net/url"

	xerrors "github.com/xbit/xbit-backend/pkg/errors"
	"github.com/xbit/xbit-backend/pkg/requestid"
)

func newProxy(route Route, logger *slog.Logger) http.Handler {
	if route.Upstream == nil {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			xerrors.WriteGraphQLError(w, r, http.StatusBadGateway, xerrors.CodeUpstreamNotConfigured, "upstream is not configured", map[string]any{
				"route": route.Name,
				"env":   route.UpstreamID,
			})
		})
	}

	target := *route.Upstream
	proxy := httputil.NewSingleHostReverseProxy(&url.URL{
		Scheme: target.Scheme,
		Host:   target.Host,
	})

	originalDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		originalHost := req.Host
		originalDirector(req)
		req.URL.Scheme = target.Scheme
		req.URL.Host = target.Host
		req.URL.Path = target.Path
		if req.URL.Path == "" {
			req.URL.Path = "/"
		}
		req.URL.RawPath = ""
		req.URL.RawQuery = mergeRawQuery(target.RawQuery, req.URL.RawQuery)
		req.Host = target.Host
		req.Header.Set("X-Forwarded-Host", originalHost)
		req.Header.Set("X-Xbit-Gateway-Route", route.Name)
		if traceID := requestid.FromContext(req.Context()); traceID != "" {
			req.Header.Set("X-Request-Id", traceID)
		}
	}

	proxy.ModifyResponse = func(resp *http.Response) error {
		resp.Header.Set("X-Xbit-Gateway-Route", route.Name)
		if traceID := requestid.FromContext(resp.Request.Context()); traceID != "" {
			resp.Header.Set("X-Request-Id", traceID)
		}
		return nil
	}

	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		logger.Error("proxy upstream error", "route", route.Name, "upstream", target.String(), "error", err, "traceId", requestid.FromContext(r.Context()))
		xerrors.WriteGraphQLError(w, r, http.StatusBadGateway, xerrors.CodeUpstreamUnavailable, "upstream unavailable", map[string]any{
			"route": route.Name,
		})
	}

	return proxy
}

func mergeRawQuery(targetQuery string, requestQuery string) string {
	switch {
	case targetQuery == "":
		return requestQuery
	case requestQuery == "":
		return targetQuery
	default:
		return targetQuery + "&" + requestQuery
	}
}
