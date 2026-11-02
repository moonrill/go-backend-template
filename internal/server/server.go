package server

import (
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/moonrill/go-backend-template/internal/config"
	"github.com/moonrill/go-backend-template/internal/middleware"
	"github.com/moonrill/go-backend-template/internal/platform/observability"
)

func New(cfg config.Config, logger *slog.Logger) (*http.Server, error) {
	switch cfg.Environment {
	case "production":
		gin.SetMode(gin.ReleaseMode)
	case "test":
		gin.SetMode(gin.TestMode)
	default:
		gin.SetMode(gin.DebugMode)
	}

	router := gin.New()
	if err := router.SetTrustedProxies(cfg.TrustedProxies); err != nil {
		return nil, fmt.Errorf("configure trusted proxies: %w", err)
	}

	router.Use(
		middleware.RequestID(),
		middleware.SecurityHeaders(),
		middleware.AccessLog(logger),
		middleware.Recovery(logger),
	)

	health := observability.NewHealthHandler()
	router.GET("/api/v1/health/live", health.Live)
	router.GET("/api/v1/health/ready", health.Ready)

	if cfg.MetricsEnabled {
		metricsHandler, err := observability.NewMetricsHandler()
		if err != nil {
			return nil, err
		}
		router.GET("/metrics", gin.WrapH(metricsHandler))
	}

	return &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           router,
		ReadTimeout:       cfg.ReadTimeout,
		WriteTimeout:      cfg.WriteTimeout,
		IdleTimeout:       cfg.IdleTimeout,
		ReadHeaderTimeout: 5 * time.Second,
	}, nil
}
