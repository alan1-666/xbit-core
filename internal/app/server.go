package app

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func RunHTTPServer(name string, addr string, handler http.Handler, logger *slog.Logger, shutdownTimeout time.Duration) {
	if shutdownTimeout <= 0 {
		shutdownTimeout = 10 * time.Second
	}

	server := &http.Server{
		Addr:              addr,
		Handler:           handler,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		logger.Info("service listening", "service", name, "addr", addr)
		errCh <- server.ListenAndServe()
	}()

	stopCh := make(chan os.Signal, 1)
	signal.Notify(stopCh, syscall.SIGINT, syscall.SIGTERM)
	select {
	case sig := <-stopCh:
		logger.Info("shutdown requested", "service", name, "signal", sig.String())
	case err := <-errCh:
		if err != nil && err != http.ErrServerClosed {
			logger.Error("server failed", "service", name, "error", err)
			os.Exit(1)
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		logger.Error("shutdown failed", "service", name, "error", err)
		os.Exit(1)
	}
}
