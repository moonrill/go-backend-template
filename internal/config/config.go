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
	Redis           RedisConfig
	RedisEnabled    bool
	MinIO           MinIOConfig
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

type RedisConfig struct {
	Enabled      bool
	URL          string
	Password     string
	DB           int
	PoolSize     int
	DialTimeout  time.Duration
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
}

type MinIOConfig struct {
	Enabled        bool
	Endpoint       string
	Region         string
	Bucket         string
	AccessKey      string
	SecretKey      string
	UseSSL         bool
	ConnectTimeout time.Duration
	AutoCreate     bool
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
		Redis: RedisConfig{
			Enabled:      false,
			URL:          getEnv("REDIS_URL", ""),
			Password:     os.Getenv("REDIS_PASSWORD"),
			DB:           0,
			PoolSize:     10,
			DialTimeout:  5 * time.Second,
			ReadTimeout:  3 * time.Second,
			WriteTimeout: 3 * time.Second,
		},
		MinIO: MinIOConfig{
			Enabled:        false,
			Endpoint:       getEnv("MINIO_ENDPOINT", ""),
			Region:         getEnv("MINIO_REGION", "us-east-1"),
			Bucket:         getEnv("MINIO_BUCKET", ""),
			AccessKey:      os.Getenv("MINIO_ACCESS_KEY"),
			SecretKey:      os.Getenv("MINIO_SECRET_KEY"),
			UseSSL:         true,
			ConnectTimeout: 5 * time.Second,
			AutoCreate:     true,
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
	if cfg.Redis.Enabled, err = getBool("REDIS_ENABLED", false); err != nil {
		return Config{}, err
	}
	cfg.RedisEnabled = cfg.Redis.Enabled
	if cfg.Redis.DB, err = getInt("REDIS_DB", cfg.Redis.DB); err != nil {
		return Config{}, err
	}
	if cfg.Redis.PoolSize, err = getInt("REDIS_POOL_SIZE", cfg.Redis.PoolSize); err != nil {
		return Config{}, err
	}
	if connTimeout := os.Getenv("REDIS_CONNECT_TIMEOUT"); connTimeout != "" && os.Getenv("REDIS_DIAL_TIMEOUT") == "" {
		if cfg.Redis.DialTimeout, err = getDuration("REDIS_CONNECT_TIMEOUT", cfg.Redis.DialTimeout); err != nil {
			return Config{}, err
		}
	} else {
		if cfg.Redis.DialTimeout, err = getDuration("REDIS_DIAL_TIMEOUT", cfg.Redis.DialTimeout); err != nil {
			return Config{}, err
		}
	}
	if cfg.Redis.ReadTimeout, err = getDuration("REDIS_READ_TIMEOUT", cfg.Redis.ReadTimeout); err != nil {
		return Config{}, err
	}
	if cfg.Redis.WriteTimeout, err = getDuration("REDIS_WRITE_TIMEOUT", cfg.Redis.WriteTimeout); err != nil {
		return Config{}, err
	}
	if cfg.MinIO.Enabled, err = getBool("MINIO_ENABLED", false); err != nil {
		return Config{}, err
	}
	cfg.MinIOEnabled = cfg.MinIO.Enabled
	if cfg.MinIO.UseSSL, err = getBool("MINIO_USE_SSL", cfg.MinIO.UseSSL); err != nil {
		return Config{}, err
	}
	if cfg.MinIO.AutoCreate, err = getBool("MINIO_AUTO_CREATE_BUCKET", cfg.MinIO.AutoCreate); err != nil {
		return Config{}, err
	}
	if cfg.MinIO.ConnectTimeout, err = getDuration("MINIO_CONNECT_TIMEOUT", cfg.MinIO.ConnectTimeout); err != nil {
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
	if err := validateRedisConfig(cfg.Redis); err != nil {
		return Config{}, err
	}
	if err := validateMinIOConfig(cfg.MinIO); err != nil {
		return Config{}, err
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

func validateRedisConfig(r RedisConfig) error {
	if !r.Enabled {
		return nil
	}
	if r.URL == "" {
		return fmt.Errorf("REDIS_URL is required when REDIS_ENABLED=true")
	}
	u, err := url.Parse(r.URL)
	if err != nil || (u.Scheme != "redis" && u.Scheme != "rediss" && u.Scheme != "unix") {
		return fmt.Errorf("REDIS_URL must be a valid redis:// or rediss:// URI")
	}
	if u.Scheme == "unix" {
		if u.Path == "" {
			return fmt.Errorf("REDIS_URL must have a path for unix scheme")
		}
	} else if u.Host == "" {
		return fmt.Errorf("REDIS_URL must be a valid redis:// or rediss:// URI")
	}
	if r.DB < 0 {
		return fmt.Errorf("REDIS_DB must not be negative")
	}
	if r.PoolSize < 1 {
		return fmt.Errorf("REDIS_POOL_SIZE must be at least 1")
	}
	if r.DialTimeout <= 0 {
		return fmt.Errorf("REDIS_DIAL_TIMEOUT must be positive")
	}
	if r.ReadTimeout <= 0 {
		return fmt.Errorf("REDIS_READ_TIMEOUT must be positive")
	}
	if r.WriteTimeout <= 0 {
		return fmt.Errorf("REDIS_WRITE_TIMEOUT must be positive")
	}
	return nil
}

func validateMinIOConfig(m MinIOConfig) error {
	if !m.Enabled {
		return nil
	}
	if m.Endpoint == "" {
		return fmt.Errorf("MINIO_ENDPOINT is required when MINIO_ENABLED=true")
	}
	if strings.HasPrefix(m.Endpoint, "http://") || strings.HasPrefix(m.Endpoint, "https://") {
		return fmt.Errorf("MINIO_ENDPOINT must not include http:// or https:// scheme prefix; use MINIO_USE_SSL to toggle TLS")
	}
	if m.Bucket == "" {
		return fmt.Errorf("MINIO_BUCKET is required when MINIO_ENABLED=true")
	}
	if err := validateBucketName(m.Bucket); err != nil {
		return fmt.Errorf("MINIO_BUCKET %w", err)
	}
	if m.AccessKey == "" {
		return fmt.Errorf("MINIO_ACCESS_KEY is required when MINIO_ENABLED=true")
	}
	if m.SecretKey == "" {
		return fmt.Errorf("MINIO_SECRET_KEY is required when MINIO_ENABLED=true")
	}
	if m.ConnectTimeout <= 0 {
		return fmt.Errorf("MINIO_CONNECT_TIMEOUT must be positive")
	}
	return nil
}

func validateBucketName(bucket string) error {
	if len(bucket) < 3 || len(bucket) > 63 {
		return fmt.Errorf("name must be between 3 and 63 characters")
	}
	for i := 0; i < len(bucket); i++ {
		c := bucket[i]
		if (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '-' || c == '.' {
			continue
		}
		return fmt.Errorf("name can only contain lowercase letters, numbers, hyphens, and periods")
	}
	if bucket[0] == '-' || bucket[0] == '.' || bucket[len(bucket)-1] == '-' || bucket[len(bucket)-1] == '.' {
		return fmt.Errorf("name must start and end with a letter or number")
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

func getInt(key string, fallback int) (int, error) {
	value, ok := os.LookupEnv(key)
	if !ok {
		return fallback, nil
	}
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return fallback, nil
	}
	parsed, err := strconv.Atoi(trimmed)
	if err != nil {
		return 0, fmt.Errorf("%s must be a valid integer", key)
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
