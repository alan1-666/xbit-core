package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/go-chi/chi/v5"

	"github.com/xbit/xbit-backend/internal/app"
	"github.com/xbit/xbit-backend/internal/config"
	"github.com/xbit/xbit-backend/internal/db"
	"github.com/xbit/xbit-backend/internal/health"
	"github.com/xbit/xbit-backend/internal/httpx"
	"github.com/xbit/xbit-backend/internal/trading"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	cfg, err := config.LoadService("trading", ":8083")
	if err != nil {
		logger.Error("load config", "error", err)
		os.Exit(1)
	}

	store := trading.Store(trading.NewMemoryStore())
	if cfg.DatabaseDSN != "" {
		postgres, err := db.ConnectPostgres(context.Background(), cfg.DatabaseDSN)
		if err != nil {
			logger.Error("connect postgres", "error", err)
			os.Exit(1)
		}
		defer postgres.Close()
		store = trading.NewPostgresStore(postgres.Pool)
	}

	router := chi.NewRouter()
	router.Use(httpx.RequestID)
	router.Use(httpx.Recoverer(logger))
	router.Use(httpx.AccessLog(logger))
	health.Register(router, cfg.Name)
	trading.NewHandler(trading.NewService(store)).RegisterRoutes(router)

	app.RunHTTPServer(cfg.Name, cfg.Addr, router, logger, cfg.ShutdownTimeout)
}
