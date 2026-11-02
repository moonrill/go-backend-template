package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/moonrill/go-backend-template/internal/config"
	"github.com/moonrill/go-backend-template/internal/platform/postgres"
	"github.com/moonrill/go-backend-template/internal/server"
)

func main() {
	defaultLogger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(defaultLogger)

	cfg, err := config.Load()
	if err != nil {
		defaultLogger.Error("configuration error", "error", err)
		os.Exit(1)
	}

	logger := newLogger(cfg.LogLevel)
	slog.SetDefault(logger)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	var deps server.Dependencies
	if cfg.Database.Enabled {
		dbPool, err := postgres.NewPool(ctx, cfg.Database)
		if err != nil {
			logger.Error("database initialization failed", "error", err)
			os.Exit(1)
		}
		defer func() {
			dbPool.Close()
			logger.Info("database connection pool closed")
		}()
		deps.DB = dbPool
		logger.Info("database connection pool initialized", "max_conns", cfg.Database.MaxConns)
	}

	srv, err := server.New(cfg, logger, deps)
	if err != nil {
		logger.Error("server initialization failed", "error", err)
		os.Exit(1)
	}

	errCh := make(chan error, 1)
	go func() {
		logger.Info("http server starting", "port", cfg.Port, "environment", cfg.Environment)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	select {
	case <-ctx.Done():
		logger.Info("shutdown signal received")
	case err := <-errCh:
		logger.Error("http server failed", "error", err)
		os.Exit(1)
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Error("graceful shutdown failed", "error", err)
		_ = srv.Close()
		os.Exit(1)
	}
	logger.Info("server stopped")
}

func newLogger(level string) *slog.Logger {
	var l slog.Level
	switch level {
	case "debug":
		l = slog.LevelDebug
	case "warn":
		l = slog.LevelWarn
	case "error":
		l = slog.LevelError
	default:
		l = slog.LevelInfo
	}
	return slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: l}))
}
