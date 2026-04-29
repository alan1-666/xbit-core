package gateway

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"

	"github.com/xbit/xbit-backend/internal/health"
	"github.com/xbit/xbit-backend/internal/httpx"
)

func NewRouter(cfg Config, logger *slog.Logger) http.Handler {
	router := chi.NewRouter()
	router.Use(httpx.RequestID)
	router.Use(httpx.Recoverer(logger))
	router.Use(httpx.AccessLog(logger))
	router.Use(cors.Handler(cors.Options{
		AllowedOrigins:   cfg.AllowedOrigins,
		AllowedMethods:   []string{http.MethodGet, http.MethodPost, http.MethodOptions},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token", "X-Consumer-Username", "X-Request-Id"},
		ExposedHeaders:   []string{"X-Request-Id", "X-Xbit-Gateway-Route"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	health.Register(router, "gateway")

	for _, route := range cfg.Routes {
		handler := http.TimeoutHandler(newProxy(route, logger), cfg.RequestTimeout, "gateway timeout")
		router.Handle(route.Path, handler)
		router.Handle(route.Path+"/", handler)
	}

	return router
}
