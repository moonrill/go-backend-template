package config

import (
	"testing"
	"time"
)

func clearEnv(t *testing.T) {
	t.Helper()
	t.Setenv("APP_NAME", "")
	t.Setenv("APP_ENV", "")
	t.Setenv("APP_PORT", "")
	t.Setenv("APP_READ_TIMEOUT", "")
	t.Setenv("APP_WRITE_TIMEOUT", "")
	t.Setenv("APP_IDLE_TIMEOUT", "")
	t.Setenv("APP_SHUTDOWN_TIMEOUT", "")
	t.Setenv("LOG_LEVEL", "")
	t.Setenv("METRICS_ENABLED", "")
	t.Setenv("TRACING_ENABLED", "")
	t.Setenv("DATABASE_ENABLED", "")
	t.Setenv("REDIS_ENABLED", "")
	t.Setenv("MINIO_ENABLED", "")
	t.Setenv("APP_TRUSTED_PROXIES", "")
}

func TestLoadDefaults(t *testing.T) {
	clearEnv(t)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.Name != "go-backend-template" {
		t.Fatalf("Name = %q, want go-backend-template", cfg.Name)
	}
	if cfg.Environment != "development" {
		t.Fatalf("Environment = %q, want development", cfg.Environment)
	}
	if cfg.Port != "8080" {
		t.Fatalf("Port = %q, want 8080", cfg.Port)
	}
	if cfg.ReadTimeout != 10*time.Second {
		t.Fatalf("ReadTimeout = %v, want 10s", cfg.ReadTimeout)
	}
	if !cfg.MetricsEnabled {
		t.Fatalf("MetricsEnabled = false, want true")
	}
	if len(cfg.TrustedProxies) != 0 {
		t.Fatalf("TrustedProxies = %v, want empty", cfg.TrustedProxies)
	}
}

func TestLoadRejectsInvalidDuration(t *testing.T) {
	clearEnv(t)
	t.Setenv("APP_READ_TIMEOUT", "not-a-duration")
	if _, err := Load(); err == nil {
		t.Fatal("expected error for invalid duration")
	}
}

func TestLoadRejectsInvalidBool(t *testing.T) {
	clearEnv(t)
	t.Setenv("METRICS_ENABLED", "not-a-bool")
	if _, err := Load(); err == nil {
		t.Fatal("expected error for invalid boolean")
	}
}

func TestLoadRejectsInvalidPort(t *testing.T) {
	for _, port := range []string{"abc", "0", "-80", "70000"} {
		t.Run(port, func(t *testing.T) {
			clearEnv(t)
			t.Setenv("APP_PORT", port)
			if _, err := Load(); err == nil {
				t.Fatalf("expected error for port %q", port)
			}
		})
	}
}

func TestLoadRejectsInvalidEnvironment(t *testing.T) {
	clearEnv(t)
	t.Setenv("APP_ENV", "invalid-env")
	if _, err := Load(); err == nil {
		t.Fatal("expected error for invalid APP_ENV")
	}
}

func TestLoadRejectsInvalidLogLevel(t *testing.T) {
	clearEnv(t)
	t.Setenv("LOG_LEVEL", "verbose")
	if _, err := Load(); err == nil {
		t.Fatal("expected error for invalid LOG_LEVEL")
	}
}

func TestLoadTrustedProxies(t *testing.T) {
	clearEnv(t)
	t.Setenv("APP_TRUSTED_PROXIES", "127.0.0.1, 10.0.0.0/8")
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if len(cfg.TrustedProxies) != 2 || cfg.TrustedProxies[0] != "127.0.0.1" || cfg.TrustedProxies[1] != "10.0.0.0/8" {
		t.Fatalf("TrustedProxies = %v, want [127.0.0.1 10.0.0.0/8]", cfg.TrustedProxies)
	}
}

func TestLoadRejectsInvalidTrustedProxies(t *testing.T) {
	for _, val := range []string{"not-an-ip", "127.0.0.1,", "10.0.0.0/99"} {
		t.Run(val, func(t *testing.T) {
			clearEnv(t)
			t.Setenv("APP_TRUSTED_PROXIES", val)
			if _, err := Load(); err == nil {
				t.Fatalf("expected error for APP_TRUSTED_PROXIES=%q", val)
			}
		})
	}
}

func TestLoadRejectsUnimplementedModules(t *testing.T) {
	for _, flag := range []string{"DATABASE_ENABLED", "REDIS_ENABLED", "MINIO_ENABLED", "TRACING_ENABLED"} {
		t.Run(flag, func(t *testing.T) {
			clearEnv(t)
			t.Setenv(flag, "true")
			if _, err := Load(); err == nil {
				t.Fatalf("expected error when %s=true in Phase 1", flag)
			}
		})
	}
}
