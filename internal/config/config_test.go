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
	t.Setenv("REDIS_URL", "")
	t.Setenv("REDIS_PASSWORD", "")
	t.Setenv("REDIS_DB", "")
	t.Setenv("REDIS_POOL_SIZE", "")
	t.Setenv("REDIS_DIAL_TIMEOUT", "")
	t.Setenv("REDIS_READ_TIMEOUT", "")
	t.Setenv("REDIS_WRITE_TIMEOUT", "")
	t.Setenv("REDIS_CONNECT_TIMEOUT", "")
	t.Setenv("MINIO_ENABLED", "")
	t.Setenv("MINIO_ENDPOINT", "")
	t.Setenv("MINIO_REGION", "")
	t.Setenv("MINIO_BUCKET", "")
	t.Setenv("MINIO_ACCESS_KEY", "")
	t.Setenv("MINIO_SECRET_KEY", "")
	t.Setenv("MINIO_USE_SSL", "")
	t.Setenv("MINIO_CONNECT_TIMEOUT", "")
	t.Setenv("MINIO_AUTO_CREATE_BUCKET", "")
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
	if cfg.Redis.Enabled {
		t.Fatalf("Redis.Enabled = true, want false")
	}
	if cfg.RedisEnabled {
		t.Fatalf("RedisEnabled = true, want false")
	}
	if cfg.Redis.PoolSize != 10 {
		t.Fatalf("Redis.PoolSize = %d, want 10", cfg.Redis.PoolSize)
	}
	if cfg.Redis.DialTimeout != 5*time.Second {
		t.Fatalf("Redis.DialTimeout = %v, want 5s", cfg.Redis.DialTimeout)
	}
	if cfg.MinIO.Enabled {
		t.Fatalf("MinIO.Enabled = true, want false")
	}
	if cfg.MinIOEnabled {
		t.Fatalf("MinIOEnabled = true, want false")
	}
	if cfg.MinIO.Region != "us-east-1" {
		t.Fatalf("MinIO.Region = %q, want us-east-1", cfg.MinIO.Region)
	}
	if !cfg.MinIO.UseSSL {
		t.Fatalf("MinIO.UseSSL = false, want true")
	}
	if !cfg.MinIO.AutoCreate {
		t.Fatalf("MinIO.AutoCreate = false, want true")
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

func TestLoadRedisConfigValidation(t *testing.T) {
	t.Run("disabled by default ignores invalid values", func(t *testing.T) {
		clearEnv(t)
		t.Setenv("REDIS_ENABLED", "false")
		t.Setenv("REDIS_URL", "not-a-redis-url")
		t.Setenv("REDIS_POOL_SIZE", "-5")
		cfg, err := Load()
		if err != nil {
			t.Fatalf("expected nil error when Redis is disabled, got %v", err)
		}
		if cfg.Redis.Enabled {
			t.Fatalf("expected Redis.Enabled = false")
		}
	})

	t.Run("enabled with valid URL and custom fields", func(t *testing.T) {
		clearEnv(t)
		t.Setenv("REDIS_ENABLED", "true")
		t.Setenv("REDIS_URL", "redis://:secret@127.0.0.1:6379/2")
		t.Setenv("REDIS_PASSWORD", "override-secret")
		t.Setenv("REDIS_DB", "3")
		t.Setenv("REDIS_POOL_SIZE", "20")
		t.Setenv("REDIS_DIAL_TIMEOUT", "4s")
		t.Setenv("REDIS_READ_TIMEOUT", "2s")
		t.Setenv("REDIS_WRITE_TIMEOUT", "2s")

		cfg, err := Load()
		if err != nil {
			t.Fatalf("Load() error = %v", err)
		}
		if !cfg.Redis.Enabled || !cfg.RedisEnabled {
			t.Fatalf("expected Redis.Enabled = true")
		}
		if cfg.Redis.URL != "redis://:secret@127.0.0.1:6379/2" {
			t.Fatalf("URL = %q, want redis://:secret@127.0.0.1:6379/2", cfg.Redis.URL)
		}
		if cfg.Redis.Password != "override-secret" {
			t.Fatalf("Password = %q, want override-secret", cfg.Redis.Password)
		}
		if cfg.Redis.DB != 3 {
			t.Fatalf("DB = %d, want 3", cfg.Redis.DB)
		}
		if cfg.Redis.PoolSize != 20 {
			t.Fatalf("PoolSize = %d, want 20", cfg.Redis.PoolSize)
		}
		if cfg.Redis.DialTimeout != 4*time.Second {
			t.Fatalf("DialTimeout = %v, want 4s", cfg.Redis.DialTimeout)
		}
	})

	t.Run("enabled with fallback connect timeout", func(t *testing.T) {
		clearEnv(t)
		t.Setenv("REDIS_ENABLED", "true")
		t.Setenv("REDIS_URL", "redis://127.0.0.1:6379/0")
		t.Setenv("REDIS_CONNECT_TIMEOUT", "7s")

		cfg, err := Load()
		if err != nil {
			t.Fatalf("Load() error = %v", err)
		}
		if cfg.Redis.DialTimeout != 7*time.Second {
			t.Fatalf("DialTimeout = %v, want 7s", cfg.Redis.DialTimeout)
		}
	})

	t.Run("enabled without URL fails", func(t *testing.T) {
		clearEnv(t)
		t.Setenv("REDIS_ENABLED", "true")
		if _, err := Load(); err == nil {
			t.Fatal("expected error when REDIS_URL is empty")
		}
	})

	t.Run("enabled with invalid URL scheme fails", func(t *testing.T) {
		clearEnv(t)
		t.Setenv("REDIS_ENABLED", "true")
		t.Setenv("REDIS_URL", "http://localhost:6379")
		if _, err := Load(); err == nil {
			t.Fatal("expected error for non-redis URL")
		}
	})

	t.Run("negative DB fails", func(t *testing.T) {
		clearEnv(t)
		t.Setenv("REDIS_ENABLED", "true")
		t.Setenv("REDIS_URL", "redis://localhost:6379/0")
		t.Setenv("REDIS_DB", "-1")
		if _, err := Load(); err == nil {
			t.Fatal("expected error for negative REDIS_DB")
		}
	})

	t.Run("zero pool size fails", func(t *testing.T) {
		clearEnv(t)
		t.Setenv("REDIS_ENABLED", "true")
		t.Setenv("REDIS_URL", "redis://localhost:6379/0")
		t.Setenv("REDIS_POOL_SIZE", "0")
		if _, err := Load(); err == nil {
			t.Fatal("expected error for REDIS_POOL_SIZE < 1")
		}
	})

	t.Run("zero dial timeout fails", func(t *testing.T) {
		clearEnv(t)
		t.Setenv("REDIS_ENABLED", "true")
		t.Setenv("REDIS_URL", "redis://localhost:6379/0")
		t.Setenv("REDIS_DIAL_TIMEOUT", "0s")
		if _, err := Load(); err == nil {
			t.Fatal("expected error for non-positive REDIS_DIAL_TIMEOUT")
		}
	})
}

func TestLoadMinIOConfigValidation(t *testing.T) {
	t.Run("disabled by default ignores invalid values", func(t *testing.T) {
		clearEnv(t)
		t.Setenv("MINIO_ENABLED", "false")
		t.Setenv("MINIO_ENDPOINT", "http://invalid")
		t.Setenv("MINIO_BUCKET", "INVALID_BUCKET_NAME")
		cfg, err := Load()
		if err != nil {
			t.Fatalf("expected nil error when MinIO is disabled, got %v", err)
		}
		if cfg.MinIO.Enabled {
			t.Fatalf("expected MinIO.Enabled = false")
		}
	})

	t.Run("enabled with valid configuration", func(t *testing.T) {
		clearEnv(t)
		t.Setenv("MINIO_ENABLED", "true")
		t.Setenv("MINIO_ENDPOINT", "127.0.0.1:9000")
		t.Setenv("MINIO_REGION", "eu-central-1")
		t.Setenv("MINIO_BUCKET", "my-test-bucket")
		t.Setenv("MINIO_ACCESS_KEY", "minioadmin")
		t.Setenv("MINIO_SECRET_KEY", "minioadmin")
		t.Setenv("MINIO_USE_SSL", "false")
		t.Setenv("MINIO_CONNECT_TIMEOUT", "4s")
		t.Setenv("MINIO_AUTO_CREATE_BUCKET", "false")

		cfg, err := Load()
		if err != nil {
			t.Fatalf("Load() error = %v", err)
		}
		if !cfg.MinIO.Enabled || !cfg.MinIOEnabled {
			t.Fatalf("expected MinIO.Enabled = true")
		}
		if cfg.MinIO.Endpoint != "127.0.0.1:9000" {
			t.Fatalf("Endpoint = %q, want 127.0.0.1:9000", cfg.MinIO.Endpoint)
		}
		if cfg.MinIO.Region != "eu-central-1" {
			t.Fatalf("Region = %q, want eu-central-1", cfg.MinIO.Region)
		}
		if cfg.MinIO.Bucket != "my-test-bucket" {
			t.Fatalf("Bucket = %q, want my-test-bucket", cfg.MinIO.Bucket)
		}
		if cfg.MinIO.AccessKey != "minioadmin" {
			t.Fatalf("AccessKey = %q, want minioadmin", cfg.MinIO.AccessKey)
		}
		if cfg.MinIO.SecretKey != "minioadmin" {
			t.Fatalf("SecretKey = %q, want minioadmin", cfg.MinIO.SecretKey)
		}
		if cfg.MinIO.UseSSL {
			t.Fatalf("expected UseSSL = false")
		}
		if cfg.MinIO.ConnectTimeout != 4*time.Second {
			t.Fatalf("ConnectTimeout = %v, want 4s", cfg.MinIO.ConnectTimeout)
		}
		if cfg.MinIO.AutoCreate {
			t.Fatalf("expected AutoCreate = false")
		}
	})

	t.Run("enabled without endpoint fails", func(t *testing.T) {
		clearEnv(t)
		t.Setenv("MINIO_ENABLED", "true")
		t.Setenv("MINIO_BUCKET", "valid-bucket")
		t.Setenv("MINIO_ACCESS_KEY", "key")
		t.Setenv("MINIO_SECRET_KEY", "secret")
		if _, err := Load(); err == nil {
			t.Fatal("expected error when MINIO_ENDPOINT is empty")
		}
	})

	t.Run("enabled with http prefix fails", func(t *testing.T) {
		clearEnv(t)
		t.Setenv("MINIO_ENABLED", "true")
		t.Setenv("MINIO_ENDPOINT", "http://127.0.0.1:9000")
		t.Setenv("MINIO_BUCKET", "valid-bucket")
		t.Setenv("MINIO_ACCESS_KEY", "key")
		t.Setenv("MINIO_SECRET_KEY", "secret")
		if _, err := Load(); err == nil {
			t.Fatal("expected error when MINIO_ENDPOINT contains http:// prefix")
		}
	})

	t.Run("enabled without bucket fails", func(t *testing.T) {
		clearEnv(t)
		t.Setenv("MINIO_ENABLED", "true")
		t.Setenv("MINIO_ENDPOINT", "127.0.0.1:9000")
		t.Setenv("MINIO_ACCESS_KEY", "key")
		t.Setenv("MINIO_SECRET_KEY", "secret")
		if _, err := Load(); err == nil {
			t.Fatal("expected error when MINIO_BUCKET is empty")
		}
	})

	t.Run("invalid bucket name fails", func(t *testing.T) {
		for _, name := range []string{"UPPERCASE", "a", "has_underscore", "-start-hyphen", "end-hyphen-"} {
			clearEnv(t)
			t.Setenv("MINIO_ENABLED", "true")
			t.Setenv("MINIO_ENDPOINT", "127.0.0.1:9000")
			t.Setenv("MINIO_BUCKET", name)
			t.Setenv("MINIO_ACCESS_KEY", "key")
			t.Setenv("MINIO_SECRET_KEY", "secret")
			if _, err := Load(); err == nil {
				t.Fatalf("expected error for invalid bucket name %q", name)
			}
		}
	})

	t.Run("enabled without access key fails", func(t *testing.T) {
		clearEnv(t)
		t.Setenv("MINIO_ENABLED", "true")
		t.Setenv("MINIO_ENDPOINT", "127.0.0.1:9000")
		t.Setenv("MINIO_BUCKET", "valid-bucket")
		t.Setenv("MINIO_SECRET_KEY", "secret")
		if _, err := Load(); err == nil {
			t.Fatal("expected error when MINIO_ACCESS_KEY is empty")
		}
	})

	t.Run("enabled without secret key fails", func(t *testing.T) {
		clearEnv(t)
		t.Setenv("MINIO_ENABLED", "true")
		t.Setenv("MINIO_ENDPOINT", "127.0.0.1:9000")
		t.Setenv("MINIO_BUCKET", "valid-bucket")
		t.Setenv("MINIO_ACCESS_KEY", "key")
		if _, err := Load(); err == nil {
			t.Fatal("expected error when MINIO_SECRET_KEY is empty")
		}
	})
}

func TestLoadRejectsUnimplementedModules(t *testing.T) {
	for _, flag := range []string{"TRACING_ENABLED"} {
		t.Run(flag, func(t *testing.T) {
			clearEnv(t)
			t.Setenv(flag, "true")
			if _, err := Load(); err == nil {
				t.Fatalf("expected error when %s=true", flag)
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
