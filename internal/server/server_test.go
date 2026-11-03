package server

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/moonrill/go-backend-template/internal/config"
	"github.com/moonrill/go-backend-template/internal/platform/redis"
	"github.com/moonrill/go-backend-template/internal/platform/storage/minio"
)

func testConfig() config.Config {
	return config.Config{
		Name:            "test-api",
		Environment:     "test",
		Port:            "8080",
		ReadTimeout:     5 * time.Second,
		WriteTimeout:    5 * time.Second,
		IdleTimeout:     30 * time.Second,
		ShutdownTimeout: 5 * time.Second,
		LogLevel:        "info",
		MetricsEnabled:  true,
		Database: config.DatabaseConfig{
			Enabled:        false,
			MaxConns:       5,
			MinConns:       1,
			ConnectTimeout: time.Second,
		},
	}
}

func TestServerHealthAndMetrics(t *testing.T) {
	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))
	cfg := testConfig()

	srv, err := New(cfg, logger)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	for _, path := range []string{"/api/v1/health/live", "/api/v1/health/ready"} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		rec := httptest.NewRecorder()
		srv.Handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("GET %s status = %d, want 200", path, rec.Code)
		}
	}

	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rec := httptest.NewRecorder()
	srv.Handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /metrics status = %d, want 200", rec.Code)
	}
}

func TestServerReadyFailsWhenDatabaseEnabledButUnavailable(t *testing.T) {
	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))
	cfg := testConfig()
	cfg.Database.Enabled = true

	srv, err := New(cfg, logger, Dependencies{DB: nil})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/health/ready", nil)
	rec := httptest.NewRecorder()
	srv.Handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("GET /api/v1/health/ready status = %d, want 503", rec.Code)
	}
}

func TestServerMetricsDisabled(t *testing.T) {
	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))
	cfg := testConfig()
	cfg.MetricsEnabled = false

	srv, err := New(cfg, logger)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rec := httptest.NewRecorder()
	srv.Handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("GET /metrics when disabled status = %d, want 404", rec.Code)
	}
}

func TestServerUntrustedProxyIgnoresForwardedFor(t *testing.T) {
	var logBuf strings.Builder
	logger := slog.New(slog.NewJSONHandler(&logBuf, nil))
	cfg := testConfig()
	cfg.TrustedProxies = nil // Do not trust any proxy by default

	srv, err := New(cfg, logger)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/health/live", nil)
	req.RemoteAddr = "192.0.2.10:12345"
	req.Header.Set("X-Forwarded-For", "203.0.113.99")
	rec := httptest.NewRecorder()
	srv.Handler.ServeHTTP(rec, req)

	logs := logBuf.String()
	if strings.Contains(logs, "203.0.113.99") {
		t.Fatalf("expected X-Forwarded-For to be ignored when TrustedProxies is empty, logs: %s", logs)
	}
	if !strings.Contains(logs, "192.0.2.10") {
		t.Fatalf("expected RemoteAddr IP 192.0.2.10 in logs, got: %s", logs)
	}
}

func TestServerRedisReadiness(t *testing.T) {
	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))

	t.Run("redis disabled does not affect readiness", func(t *testing.T) {
		cfg := testConfig()
		cfg.Redis.Enabled = false

		srv, err := New(cfg, logger)
		if err != nil {
			t.Fatalf("New() error = %v", err)
		}

		req := httptest.NewRequest(http.MethodGet, "/api/v1/health/ready", nil)
		rec := httptest.NewRecorder()
		srv.Handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("GET /api/v1/health/ready status = %d, want 200", rec.Code)
		}
	})

	t.Run("redis enabled but unavailable fails ready, live still passes", func(t *testing.T) {
		cfg := testConfig()
		cfg.Redis.Enabled = true
		cfg.Redis.DialTimeout = 500 * time.Millisecond

		srv, err := New(cfg, logger, Dependencies{Redis: nil})
		if err != nil {
			t.Fatalf("New() error = %v", err)
		}

		// Liveness must NOT fail just because Redis is unavailable
		liveReq := httptest.NewRequest(http.MethodGet, "/api/v1/health/live", nil)
		liveRec := httptest.NewRecorder()
		srv.Handler.ServeHTTP(liveRec, liveReq)
		if liveRec.Code != http.StatusOK {
			t.Fatalf("GET /api/v1/health/live status = %d, want 200", liveRec.Code)
		}

		// Readiness must reflect unavailable dependency
		readyReq := httptest.NewRequest(http.MethodGet, "/api/v1/health/ready", nil)
		readyRec := httptest.NewRecorder()
		srv.Handler.ServeHTTP(readyRec, readyReq)
		if readyRec.Code != http.StatusServiceUnavailable {
			t.Fatalf("GET /api/v1/health/ready status = %d, want 503", readyRec.Code)
		}
		if !strings.Contains(readyRec.Body.String(), `"redis":"unavailable"`) {
			t.Fatalf("expected redis:unavailable in body, got: %s", readyRec.Body.String())
		}
	})

	t.Run("redis enabled and healthy passes readiness", func(t *testing.T) {
		mr := miniredis.RunT(t)
		cfg := testConfig()
		cfg.Redis.Enabled = true
		cfg.Redis.URL = "redis://" + mr.Addr() + "/0"
		cfg.Redis.DialTimeout = 1 * time.Second

		client, err := redis.New(context.Background(), cfg.Redis)
		if err != nil {
			t.Fatalf("redis.New() error = %v", err)
		}
		defer client.Close()

		srv, err := New(cfg, logger, Dependencies{Redis: client})
		if err != nil {
			t.Fatalf("New() error = %v", err)
		}

		req := httptest.NewRequest(http.MethodGet, "/api/v1/health/ready", nil)
		rec := httptest.NewRecorder()
		srv.Handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("GET /api/v1/health/ready status = %d, want 200", rec.Code)
		}
		if !strings.Contains(rec.Body.String(), `"redis":"ready"`) {
			t.Fatalf("expected redis:ready in body, got: %s", rec.Body.String())
		}
	})
}

func TestServerMinIOReadiness(t *testing.T) {
	logger := slog.New(slog.NewJSONHandler(io.Discard, nil))

	t.Run("minio disabled does not affect readiness", func(t *testing.T) {
		cfg := testConfig()
		cfg.MinIO.Enabled = false

		srv, err := New(cfg, logger)
		if err != nil {
			t.Fatalf("New() error = %v", err)
		}

		req := httptest.NewRequest(http.MethodGet, "/api/v1/health/ready", nil)
		rec := httptest.NewRecorder()
		srv.Handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("GET /api/v1/health/ready status = %d, want 200", rec.Code)
		}
	})

	t.Run("minio enabled but unavailable fails ready, live still passes", func(t *testing.T) {
		cfg := testConfig()
		cfg.MinIO.Enabled = true
		cfg.MinIO.ConnectTimeout = 500 * time.Millisecond

		srv, err := New(cfg, logger, Dependencies{MinIO: nil})
		if err != nil {
			t.Fatalf("New() error = %v", err)
		}

		// Liveness must NOT fail just because MinIO is unavailable
		liveReq := httptest.NewRequest(http.MethodGet, "/api/v1/health/live", nil)
		liveRec := httptest.NewRecorder()
		srv.Handler.ServeHTTP(liveRec, liveReq)
		if liveRec.Code != http.StatusOK {
			t.Fatalf("GET /api/v1/health/live status = %d, want 200", liveRec.Code)
		}

		// Readiness must reflect unavailable dependency
		readyReq := httptest.NewRequest(http.MethodGet, "/api/v1/health/ready", nil)
		readyRec := httptest.NewRecorder()
		srv.Handler.ServeHTTP(readyRec, readyReq)
		if readyRec.Code != http.StatusServiceUnavailable {
			t.Fatalf("GET /api/v1/health/ready status = %d, want 503", readyRec.Code)
		}
		if !strings.Contains(readyRec.Body.String(), `"minio":"unavailable"`) {
			t.Fatalf("expected minio:unavailable in body, got: %s", readyRec.Body.String())
		}
	})

	t.Run("minio enabled and healthy passes readiness", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.Method {
			case http.MethodHead:
				w.WriteHeader(http.StatusOK)
			case http.MethodGet:
				w.Header().Set("Content-Type", "application/xml")
				_, _ = w.Write([]byte(`<LocationConstraint xmlns="http://s3.amazonaws.com/doc/2006-03-01/">us-east-1</LocationConstraint>`))
			default:
				w.WriteHeader(http.StatusOK)
			}
		}))
		defer ts.Close()

		endpoint := strings.TrimPrefix(ts.URL, "http://")
		cfg := testConfig()
		cfg.MinIO.Enabled = true
		cfg.MinIO.Endpoint = endpoint
		cfg.MinIO.Bucket = "ready-bucket"
		cfg.MinIO.AccessKey = "key"
		cfg.MinIO.SecretKey = "secret"
		cfg.MinIO.UseSSL = false
		cfg.MinIO.ConnectTimeout = 1 * time.Second

		client, err := minio.New(context.Background(), cfg.MinIO)
		if err != nil {
			t.Fatalf("minio.New() error = %v", err)
		}
		defer client.Close()

		srv, err := New(cfg, logger, Dependencies{MinIO: client})
		if err != nil {
			t.Fatalf("New() error = %v", err)
		}

		req := httptest.NewRequest(http.MethodGet, "/api/v1/health/ready", nil)
		rec := httptest.NewRecorder()
		srv.Handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("GET /api/v1/health/ready status = %d, want 200", rec.Code)
		}
		if !strings.Contains(rec.Body.String(), `"minio":"ready"`) {
			t.Fatalf("expected minio:ready in body, got: %s", rec.Body.String())
		}
	})
}
