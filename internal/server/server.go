package server

import (
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/moonrill/go-backend-template/internal/config"
	"github.com/moonrill/go-backend-template/internal/middleware"
	"github.com/moonrill/go-backend-template/internal/modules/users"
	"github.com/moonrill/go-backend-template/internal/platform/observability"
	"github.com/moonrill/go-backend-template/internal/platform/postgres"
)

// Dependencies holds optional external resources wired into the HTTP server.
type Dependencies struct {
	DB *pgxpool.Pool
}

func New(cfg config.Config, logger *slog.Logger, deps ...Dependencies) (*http.Server, error) {
	var dep Dependencies
	if len(deps) > 0 {
		dep = deps[0]
	}

	switch cfg.GinMode {
	case "release":
		gin.SetMode(gin.ReleaseMode)
	case "test":
		gin.SetMode(gin.TestMode)
	case "debug":
		gin.SetMode(gin.DebugMode)
	default:
		switch cfg.Environment {
		case "production":
			gin.SetMode(gin.ReleaseMode)
		case "test":
			gin.SetMode(gin.TestMode)
		default:
			gin.SetMode(gin.DebugMode)
		}
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

	var readinessChecks []observability.ReadinessCheck
	if cfg.Database.Enabled {
		readinessChecks = append(readinessChecks, observability.ReadinessCheck{
			Name:  "postgres",
			Check: postgres.HealthCheck(dep.DB, cfg.Database.ConnectTimeout),
		})
	}

	apiV1 := router.Group("/api/v1")

	health := observability.NewHealthHandler(readinessChecks...)
	apiV1.GET("/health/live", health.Live)
	apiV1.GET("/health/ready", health.Ready)

	if cfg.Database.Enabled && dep.DB != nil {
		usersRepo := users.NewRepository(dep.DB)
		usersService := users.NewService(usersRepo)
		usersHandler := users.NewHandler(usersService, logger)
		usersHandler.RegisterRoutes(apiV1)
	}

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
