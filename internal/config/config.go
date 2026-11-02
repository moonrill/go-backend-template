package config

import (
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Name            string
	Environment     string
	Port            string
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	IdleTimeout     time.Duration
	ShutdownTimeout time.Duration
	LogLevel        string
	MetricsEnabled  bool
	TracingEnabled  bool
	DatabaseEnabled bool
	RedisEnabled    bool
	MinIOEnabled    bool
	TrustedProxies  []string
}

func Load() (Config, error) {
	cfg := Config{
		Name:            getEnv("APP_NAME", "go-backend-template"),
		Environment:     getEnv("APP_ENV", "development"),
		Port:            getEnv("APP_PORT", "8080"),
		ReadTimeout:     10 * time.Second,
		WriteTimeout:    15 * time.Second,
		IdleTimeout:     60 * time.Second,
		ShutdownTimeout: 15 * time.Second,
		LogLevel:        getEnv("LOG_LEVEL", "info"),
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
	if cfg.DatabaseEnabled, err = getBool("DATABASE_ENABLED", false); err != nil {
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
	case "development", "staging", "production", "test":
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

	if cfg.DatabaseEnabled {
		return Config{}, fmt.Errorf("DATABASE_ENABLED is not yet supported in this phase")
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
