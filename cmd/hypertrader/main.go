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
	"github.com/xbit/xbit-backend/internal/hypertrader"
	"github.com/xbit/xbit-backend/internal/streambridge"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	cfg, err := config.LoadService("hypertrader", ":8086")
	if err != nil {
		logger.Error("load config", "error", err)
		os.Exit(1)
	}

	store := hypertrader.Store(hypertrader.NewMemoryStore())
	if cfg.DatabaseDSN != "" {
		postgres, err := db.ConnectPostgres(context.Background(), cfg.DatabaseDSN)
		if err != nil {
			logger.Error("connect postgres", "error", err)
			os.Exit(1)
		}
		defer postgres.Close()
		store = hypertrader.NewPostgresStore(postgres.Pool)
	}
	provider := hypertrader.Provider(hypertrader.NewLocalProvider())
	if cfg.HyperliquidMode == "http" {
		provider = hypertrader.NewHTTPProvider(cfg.HyperliquidURL, cfg.ProviderTimeout)
	}

	var streamSvc *streambridge.Service
	var streamCancel context.CancelFunc
	if cfg.HyperliquidWSEnabled {
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
				logger.Error("init hyperliquid mqtt publisher", "error", err)
				os.Exit(1)
			}
		}
		streamSvc = streambridge.NewService(publisher)
		defer streamSvc.Close()
		var streamCtx context.Context
		streamCtx, streamCancel = context.WithCancel(context.Background())
		defer streamCancel()
		stateStore, _ := store.(hypertrader.StateStore)
		bridge := hypertrader.NewHyperliquidStreamBridge(hypertrader.StreamBridgeConfig{
			WSURL:      cfg.HyperliquidWSURL,
			Users:      cfg.HyperliquidWSUsers,
			Dex:        cfg.HyperliquidWSDex,
			StateStore: stateStore,
			Provider:   provider,
			Logger:     logger,
		}, streamSvc)
		go bridge.Run(streamCtx)
	}

	router := chi.NewRouter()
	router.Use(httpx.RequestID)
	router.Use(httpx.Recoverer(logger))
	router.Use(httpx.AccessLog(logger))
	health.Register(router, cfg.Name)
	hypertrader.NewHandler(hypertrader.NewServiceWithProvider(store, provider)).RegisterRoutes(router)
	if streamSvc != nil {
		streambridge.NewHandler(streamSvc).RegisterRoutes(router)
	}

	app.RunHTTPServer(cfg.Name, cfg.Addr, router, logger, cfg.ShutdownTimeout)
	if streamCancel != nil {
		streamCancel()
	}
}
