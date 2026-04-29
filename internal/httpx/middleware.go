package httpx

import (
	"log/slog"
	"net/http"
	"runtime/debug"
	"time"

	xerrors "github.com/xbit/xbit-backend/pkg/errors"
	"github.com/xbit/xbit-backend/pkg/requestid"
)

type statusRecorder struct {
	http.ResponseWriter
	status int
	bytes  int
}

func (r *statusRecorder) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}

func (r *statusRecorder) Write(b []byte) (int, error) {
	if r.status == 0 {
		r.status = http.StatusOK
	}
	n, err := r.ResponseWriter.Write(b)
	r.bytes += n
	return n, err
}

func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get("X-Request-Id")
		if id == "" {
			id = requestid.New()
		}
		w.Header().Set("X-Request-Id", id)
		next.ServeHTTP(w, r.WithContext(requestid.WithContext(r.Context(), id)))
	})
}

func AccessLog(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			rec := &statusRecorder{ResponseWriter: w}
			next.ServeHTTP(rec, r)
			status := rec.status
			if status == 0 {
				status = http.StatusOK
			}
			logger.Info("http request",
				"method", r.Method,
				"path", r.URL.Path,
				"query", r.URL.RawQuery,
				"status", status,
				"bytes", rec.bytes,
				"durationMs", time.Since(start).Milliseconds(),
				"traceId", requestid.FromContext(r.Context()),
				"remoteAddr", r.RemoteAddr,
				"userAgent", r.UserAgent(),
			)
		})
	}
}

func Recoverer(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if recovered := recover(); recovered != nil {
					logger.Error("panic recovered",
						"panic", recovered,
						"stack", string(debug.Stack()),
						"traceId", requestid.FromContext(r.Context()),
					)
					xerrors.WriteGraphQLError(w, r, http.StatusInternalServerError, xerrors.CodeInternal, "internal server error", nil)
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}
