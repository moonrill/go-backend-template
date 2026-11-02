package server

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/moonrill/go-backend-template/internal/config"
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

