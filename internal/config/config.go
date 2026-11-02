package config

import (
	"bufio"
	"errors"
	"fmt"
	"net"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Name            string
	Environment     string
	GinMode         string
	Port            string
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	IdleTimeout     time.Duration
	ShutdownTimeout time.Duration
	LogLevel        string
	MetricsEnabled  bool
	TracingEnabled  bool
	Database        DatabaseConfig
	RedisEnabled    bool
	MinIOEnabled    bool
	TrustedProxies  []string
}

type DatabaseConfig struct {
	Enabled           bool
	URL               string
	MaxConns          int32
	MinConns          int32
	ConnectTimeout    time.Duration
	MaxConnLifetime   time.Duration
	MaxConnIdleTime   time.Duration
	HealthCheckPeriod time.Duration
}

func Load() (Config, error) {
	if err := loadDotEnv(".env"); err != nil {
		return Config{}, err
	}

	cfg := Config{
		Name:            getEnv("APP_NAME", "go-backend-template"),
		Environment:     getEnv("APP_ENV", "development"),
		Port:            getEnv("APP_PORT", "8080"),
		ReadTimeout:     10 * time.Second,
		WriteTimeout:    15 * time.Second,
		IdleTimeout:     60 * time.Second,
		ShutdownTimeout: 15 * time.Second,
		LogLevel:        getEnv("LOG_LEVEL", "info"),
		Database: DatabaseConfig{
			URL:               getEnv("DATABASE_URL", ""),
			MaxConns:          10,
			MinConns:          1,
			ConnectTimeout:    3 * time.Second,
			MaxConnLifetime:   30 * time.Minute,
			MaxConnIdleTime:   5 * time.Minute,
			HealthCheckPeriod: 1 * time.Minute,
		},
	}

	var err error
	if cfg.ReadTimeout, err = getDuration("APP_READ_TIMEOUT", cfg.ReadTimeout); err != nil {
		return Config{}, err
	}
	if cfg.WriteTimeout, err = getDuration("APP_WRITE_TIMEOUT", cfg.WriteTimeout); err != nil {
		return Config{}, err
	}
	if cfg.IdleTimeout, err = getDuration("APP_IDLE_TIMEOUT", cfg.IdleTimeout); err != nil {
		return Config{}, err
	}
	if cfg.ShutdownTimeout, err = getDuration("APP_SHUTDOWN_TIMEOUT", cfg.ShutdownTimeout); err != nil {
		return Config{}, err
	}
	if cfg.MetricsEnabled, err = getBool("METRICS_ENABLED", true); err != nil {
		return Config{}, err
	}
	if cfg.TracingEnabled, err = getBool("TRACING_ENABLED", false); err != nil {
		return Config{}, err
	}
	if cfg.Database.Enabled, err = getBool("DATABASE_ENABLED", false); err != nil {
		return Config{}, err
	}
	if cfg.Database.MaxConns, err = getInt32("DATABASE_MAX_CONNS", cfg.Database.MaxConns); err != nil {
		return Config{}, err
	}
	if cfg.Database.MinConns, err = getInt32("DATABASE_MIN_CONNS", cfg.Database.MinConns); err != nil {
		return Config{}, err
	}
	if cfg.Database.ConnectTimeout, err = getDuration("DATABASE_CONNECT_TIMEOUT", cfg.Database.ConnectTimeout); err != nil {
		return Config{}, err
	}
	if cfg.Database.MaxConnLifetime, err = getDuration("DATABASE_MAX_CONN_LIFETIME", cfg.Database.MaxConnLifetime); err != nil {
		return Config{}, err
	}
	if cfg.Database.MaxConnIdleTime, err = getDuration("DATABASE_MAX_CONN_IDLE_TIME", cfg.Database.MaxConnIdleTime); err != nil {
		return Config{}, err
	}
	if cfg.Database.HealthCheckPeriod, err = getDuration("DATABASE_HEALTH_CHECK_PERIOD", cfg.Database.HealthCheckPeriod); err != nil {
		return Config{}, err
	}
	if cfg.RedisEnabled, err = getBool("REDIS_ENABLED", false); err != nil {
		return Config{}, err
	}
	if cfg.MinIOEnabled, err = getBool("MINIO_ENABLED", false); err != nil {
		return Config{}, err
	}
	if cfg.TrustedProxies, err = getTrustedProxies("APP_TRUSTED_PROXIES"); err != nil {
		return Config{}, err
	}

	if cfg.Name == "" {
		return Config{}, fmt.Errorf("APP_NAME must not be empty")
	}
	switch cfg.Environment {
	case "production":
		cfg.GinMode = "release"
	case "test":
		cfg.GinMode = "test"
	case "development", "staging":
		cfg.GinMode = "debug"
	default:
		return Config{}, fmt.Errorf("APP_ENV must be development, staging, production, or test")
	}
	if err := validatePort(cfg.Port); err != nil {
		return Config{}, err
	}
	switch cfg.LogLevel {
	case "debug", "info", "warn", "error":
	default:
		return Config{}, fmt.Errorf("LOG_LEVEL must be debug, info, warn, or error")
	}

	if err := validateDatabaseConfig(cfg.Database); err != nil {
		return Config{}, err
	}
	if cfg.RedisEnabled {
		return Config{}, fmt.Errorf("REDIS_ENABLED is not yet supported in this phase")
	}
	if cfg.MinIOEnabled {
		return Config{}, fmt.Errorf("MINIO_ENABLED is not yet supported in this phase")
	}
	if cfg.TracingEnabled {
		return Config{}, fmt.Errorf("TRACING_ENABLED is not yet supported in this phase")
	}

	return cfg, nil
}

func validateDatabaseConfig(db DatabaseConfig) error {
	if db.MaxConns < 1 {
		return fmt.Errorf("DATABASE_MAX_CONNS must be at least 1")
	}
	if db.MinConns < 0 {
		return fmt.Errorf("DATABASE_MIN_CONNS must not be negative")
	}
	if db.MinConns > db.MaxConns {
		return fmt.Errorf("DATABASE_MIN_CONNS must not exceed DATABASE_MAX_CONNS")
	}
	if !db.Enabled {
		return nil
	}
	if db.URL == "" {
		return fmt.Errorf("DATABASE_URL is required when DATABASE_ENABLED=true")
	}
	u, err := url.Parse(db.URL)
	if err != nil || (u.Scheme != "postgres" && u.Scheme != "postgresql") || u.Host == "" {
		return fmt.Errorf("DATABASE_URL must be a valid postgres:// or postgresql:// URI")
	}
	return nil
}

func getEnv(key, fallback string) string {
	value, ok := os.LookupEnv(key)
	if !ok {
		return fallback
	}
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return fallback
	}
	return trimmed
}

func getBool(key string, fallback bool) (bool, error) {
	value, ok := os.LookupEnv(key)
	if !ok {
		return fallback, nil
	}
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return fallback, nil
	}
	parsed, err := strconv.ParseBool(trimmed)
	if err != nil {
		return false, fmt.Errorf("%s must be a valid boolean", key)
	}
	return parsed, nil
}

func getInt32(key string, fallback int32) (int32, error) {
	value, ok := os.LookupEnv(key)
	if !ok {
		return fallback, nil
	}
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return fallback, nil
	}
	parsed, err := strconv.ParseInt(trimmed, 10, 32)
	if err != nil {
		return 0, fmt.Errorf("%s must be a valid 32-bit integer", key)
	}
	return int32(parsed), nil
}

func getDuration(key string, fallback time.Duration) (time.Duration, error) {
	value, ok := os.LookupEnv(key)
	if !ok {
		return fallback, nil
	}
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return fallback, nil
	}
	parsed, err := time.ParseDuration(trimmed)
	if err != nil || parsed <= 0 {
		return 0, fmt.Errorf("%s must be a valid positive duration", key)
	}
	return parsed, nil
}

func validatePort(port string) error {
	p, err := strconv.Atoi(port)
	if err != nil || p < 1 || p > 65535 {
		return fmt.Errorf("APP_PORT must be an integer between 1 and 65535")
	}
	return nil
}

func getTrustedProxies(key string) ([]string, error) {
	value, ok := os.LookupEnv(key)
	if !ok {
		return nil, nil
	}
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil, nil
	}
	parts := strings.Split(trimmed, ",")
	proxies := make([]string, 0, len(parts))
	for _, part := range parts {
		p := strings.TrimSpace(part)
		if p == "" {
			return nil, fmt.Errorf("%s contains an empty entry", key)
		}
		if net.ParseIP(p) == nil {
			if _, _, err := net.ParseCIDR(p); err != nil {
				return nil, fmt.Errorf("%s entry %q must be a valid IP address or CIDR range", key, p)
			}
		}
		proxies = append(proxies, p)
	}
	return proxies, nil
}

// loadDotEnv reads KEY=VALUE pairs from path if the file exists and sets any
// environment variables that are not already present in the process environment.
func loadDotEnv(path string) error {
	f, err := os.Open(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("open %s: %w", path, err)
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		line = strings.TrimPrefix(line, "export ")
		key, val, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		val = strings.TrimSpace(val)
		if len(val) >= 2 {
			if (val[0] == '"' && val[len(val)-1] == '"') || (val[0] == '\'' && val[len(val)-1] == '\'') {
				val = val[1 : len(val)-1]
			}
		}
		if key == "" {
			continue
		}
		if _, exists := os.LookupEnv(key); !exists {
			_ = os.Setenv(key, val)
		}
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("read %s: %w", path, err)
	}
	return nil
}
