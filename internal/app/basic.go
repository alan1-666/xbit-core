package app

import (
	"log/slog"
	"os"

	"github.com/go-chi/chi/v5"

	"github.com/xbit/xbit-backend/internal/health"
	"github.com/xbit/xbit-backend/internal/httpx"
)

// RunBasicService starts the minimal HTTP surface every service must expose.
// Business handlers are added as each service leaves scaffold mode.
func RunBasicService(name string, defaultAddr string) {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	addr := os.Getenv("SERVICE_ADDR")
	if addr == "" {
		addr = defaultAddr
	}

	router := chi.NewRouter()
	router.Use(httpx.RequestID)
	router.Use(httpx.Recoverer(logger))
	router.Use(httpx.AccessLog(logger))
	health.Register(router, name)

	RunHTTPServer(name, addr, router, logger, 0)
}
