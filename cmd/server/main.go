package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"camp-scheduler/internal/config"
	"camp-scheduler/internal/server"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		slog.With("error", err).Error("error loading config")
		os.Exit(1)
	}

	initLogger(cfg)

	srv, err := server.New(cfg)
	if err != nil {
		slog.With("error", err).Error("error initializing server")
		os.Exit(1)
	}

	httpServer := &http.Server{
		Addr:         srv.Addr(),
		Handler:      srv,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Graceful shutdown
	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGTERM)

	go func() {
		slog.With("addr", httpServer.Addr).Info("starting server")
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.With("error", err).Error("error starting server")
			os.Exit(1)
		}
	}()

	<-done
	slog.Info("shutting down server")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := httpServer.Shutdown(ctx); err != nil {
		slog.With("error", err).Error("error shutting down server")
	}
	srv.Shutdown()

	slog.Info("server stopped")
}

func initLogger(cfg config.Config) {
	var handler slog.Handler

	opts := &slog.HandlerOptions{}
	switch cfg.Server.LogLevel {
	case "debug":
		opts.Level = slog.LevelDebug
	case "warn":
		opts.Level = slog.LevelWarn
	case "error":
		opts.Level = slog.LevelError
	default:
		opts.Level = slog.LevelInfo
	}

	if cfg.Server.Mode == "local" {
		handler = slog.NewTextHandler(os.Stderr, opts)
	} else {
		handler = slog.NewJSONHandler(os.Stderr, opts)
	}

	slog.SetDefault(slog.New(handler))
}
