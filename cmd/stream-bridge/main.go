package main

import (
	"log/slog"
	"os"

	"github.com/go-chi/chi/v5"

	"github.com/xbit/xbit-backend/internal/app"
	"github.com/xbit/xbit-backend/internal/config"
	"github.com/xbit/xbit-backend/internal/health"
	"github.com/xbit/xbit-backend/internal/httpx"
	"github.com/xbit/xbit-backend/internal/streambridge"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	cfg, err := config.LoadService("stream-bridge", ":8085")
	if err != nil {
		logger.Error("load config", "error", err)
		os.Exit(1)
	}

	var publisher streambridge.Publisher = streambridge.NewMemoryPublisher()
	if cfg.MQTTEnabled {
		publisher, err = streambridge.NewMQTTPublisher(streambridge.Config{
			BrokerURL: cfg.MQTTBrokerURL,
			ClientID:  cfg.MQTTClientID,
			Username:  cfg.MQTTUsername,
			Password:  cfg.MQTTPassword,
			Enabled:   cfg.MQTTEnabled,
		}, logger)
		if err != nil {
			logger.Error("init mqtt publisher", "error", err)
			os.Exit(1)
		}
	}

	service := streambridge.NewService(publisher)
	defer service.Close()

	router := chi.NewRouter()
	router.Use(httpx.RequestID)
	router.Use(httpx.Recoverer(logger))
	router.Use(httpx.AccessLog(logger))
	health.Register(router, cfg.Name)
	streambridge.NewHandler(service).RegisterRoutes(router)

	app.RunHTTPServer(cfg.Name, cfg.Addr, router, logger, cfg.ShutdownTimeout)
}
