package config

import (
	"os"
	"path/filepath"
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
	t.Setenv("DATABASE_URL", "")
	t.Setenv("DATABASE_MAX_CONNS", "")
	t.Setenv("DATABASE_MIN_CONNS", "")
	t.Setenv("DATABASE_CONNECT_TIMEOUT", "")
	t.Setenv("DATABASE_MAX_CONN_LIFETIME", "")
	t.Setenv("DATABASE_MAX_CONN_IDLE_TIME", "")
	t.Setenv("DATABASE_HEALTH_CHECK_PERIOD", "")
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
	if cfg.Database.Enabled {
		t.Fatalf("Database.Enabled = true, want false")
	}
	if cfg.Database.MaxConns != 10 || cfg.Database.MinConns != 1 {
		t.Fatalf("Database Conns = (%d, %d), want (10, 1)", cfg.Database.MaxConns, cfg.Database.MinConns)
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

func TestLoadGinModeFromAppEnv(t *testing.T) {
	tests := []struct {
		appEnv      string
		wantGinMode string
	}{
		{"development", "debug"},
		{"staging", "debug"},
		{"production", "release"},
		{"test", "test"},
	}

	for _, tc := range tests {
		t.Run(tc.appEnv, func(t *testing.T) {
			clearEnv(t)
			t.Setenv("APP_ENV", tc.appEnv)
			cfg, err := Load()
			if err != nil {
				t.Fatalf("Load() error = %v", err)
			}
			if cfg.GinMode != tc.wantGinMode {
				t.Fatalf("GinMode = %q, want %q", cfg.GinMode, tc.wantGinMode)
			}
		})
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

func TestLoadDatabaseConfigValidation(t *testing.T) {
	t.Run("enabled with valid URL", func(t *testing.T) {
		clearEnv(t)
		t.Setenv("DATABASE_ENABLED", "true")
		t.Setenv("DATABASE_URL", "postgres://user:pass@localhost:5432/appdb?sslmode=disable")
		t.Setenv("DATABASE_MAX_CONNS", "20")
		t.Setenv("DATABASE_MIN_CONNS", "2")
		cfg, err := Load()
		if err != nil {
			t.Fatalf("Load() error = %v", err)
		}
		if !cfg.Database.Enabled || cfg.Database.MaxConns != 20 || cfg.Database.MinConns != 2 {
			t.Fatalf("unexpected database config: %+v", cfg.Database)
		}
	})

	t.Run("enabled without URL fails", func(t *testing.T) {
		clearEnv(t)
		t.Setenv("DATABASE_ENABLED", "true")
		if _, err := Load(); err == nil {
			t.Fatal("expected error when DATABASE_ENABLED=true and DATABASE_URL is empty")
		}
	})

	t.Run("enabled with invalid URL scheme fails", func(t *testing.T) {
		clearEnv(t)
		t.Setenv("DATABASE_ENABLED", "true")
		t.Setenv("DATABASE_URL", "mysql://localhost:3306/db")
		if _, err := Load(); err == nil {
			t.Fatal("expected error for non-postgres DATABASE_URL")
		}
	})

	t.Run("min conns exceeding max conns fails", func(t *testing.T) {
		clearEnv(t)
		t.Setenv("DATABASE_MAX_CONNS", "5")
		t.Setenv("DATABASE_MIN_CONNS", "10")
		if _, err := Load(); err == nil {
			t.Fatal("expected error when DATABASE_MIN_CONNS > DATABASE_MAX_CONNS")
		}
	})

	t.Run("zero max conns fails", func(t *testing.T) {
		clearEnv(t)
		t.Setenv("DATABASE_MAX_CONNS", "0")
		if _, err := Load(); err == nil {
			t.Fatal("expected error when DATABASE_MAX_CONNS < 1")
		}
	})
}

func TestLoadRejectsUnimplementedModules(t *testing.T) {
	for _, flag := range []string{"REDIS_ENABLED", "MINIO_ENABLED", "TRACING_ENABLED"} {
		t.Run(flag, func(t *testing.T) {
			clearEnv(t)
			t.Setenv(flag, "true")
			if _, err := Load(); err == nil {
				t.Fatalf("expected error when %s=true in Phase 2", flag)
			}
		})
	}
}

func TestLoadDotEnvFile(t *testing.T) {
	tmpDir := t.TempDir()
	envFile := filepath.Join(tmpDir, ".env")
	content := "APP_ENV=production\nAPP_PORT=9090\n# comment line\n"
	if err := os.WriteFile(envFile, []byte(content), 0o600); err != nil {
		t.Fatalf("write temp .env: %v", err)
	}

	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(origDir)
	})

	// Unset APP_ENV and APP_PORT so loadDotEnv populates them from .env
	_ = os.Unsetenv("APP_ENV")
	_ = os.Unsetenv("APP_PORT")
	t.Cleanup(func() {
		_ = os.Unsetenv("APP_ENV")
		_ = os.Unsetenv("APP_PORT")
	})

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.Environment != "production" || cfg.GinMode != "release" || cfg.Port != "9090" {
		t.Fatalf("got Environment=%q GinMode=%q Port=%q, want production/release/9090", cfg.Environment, cfg.GinMode, cfg.Port)
	}
}
