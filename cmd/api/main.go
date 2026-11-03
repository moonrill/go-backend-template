package main

// @title Go Backend Template API
// @version 1.0
// @description Lightweight, modular Go REST API starter built with Gin, PostgreSQL, Redis, MinIO, and OpenTelemetry.
// @termsOfService http://swagger.io/terms/

// @contact.name API Support
// @contact.url https://github.com/moonrill/go-backend-template

// @license.name MIT
// @license.url https://opensource.org/licenses/MIT

// @host localhost:8080
// @BasePath /api/v1
// @schemes http https

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/moonrill/go-backend-template/internal/config"
	"github.com/moonrill/go-backend-template/internal/otel"
	"github.com/moonrill/go-backend-template/internal/platform/postgres"
	"github.com/moonrill/go-backend-template/internal/platform/redis"
	"github.com/moonrill/go-backend-template/internal/platform/storage/minio"
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

	otelShutdown, err := otel.Init(ctx, time.Local)
	if err != nil {
		logger.Error("tracing initialization failed", "error", err)
		os.Exit(1)
	}
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := otelShutdown(shutdownCtx); err != nil {
			logger.Warn("tracing shutdown failed", "error", err)
		}
	}()

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

	if cfg.Redis.Enabled {
		redisClient, err := redis.New(ctx, cfg.Redis)
		if err != nil {
			logger.Error("redis initialization failed", "error", err)
			os.Exit(1)
		}
		defer func() {
			if err := redisClient.Close(); err != nil {
				logger.Warn("redis close failed", "error", err)
			} else {
				logger.Info("redis client closed")
			}
		}()
		deps.Redis = redisClient
		logger.Info("redis client initialized", "db", cfg.Redis.DB, "pool_size", cfg.Redis.PoolSize)
	}

	if cfg.MinIO.Enabled {
		minioClient, err := minio.New(ctx, cfg.MinIO)
		if err != nil {
			logger.Error("minio initialization failed", "error", err)
			os.Exit(1)
		}
		defer func() {
			if err := minioClient.Close(); err != nil {
				logger.Warn("minio close failed", "error", err)
			} else {
				logger.Info("minio client closed")
			}
		}()
		deps.MinIO = minioClient
		logger.Info("minio client initialized", "endpoint", cfg.MinIO.Endpoint, "bucket", cfg.MinIO.Bucket, "use_ssl", cfg.MinIO.UseSSL)
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
