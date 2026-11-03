package redis

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/moonrill/go-backend-template/internal/config"
	goredis "github.com/redis/go-redis/v9"
)

// Client wraps the underlying Redis client with lifecycle and health helpers.
type Client struct {
	rdb *goredis.Client
}

// New creates and validates a new Redis connection client based on the provided configuration.
func New(ctx context.Context, cfg config.RedisConfig) (*Client, error) {
	if !cfg.Enabled {
		return nil, errors.New("cannot initialize redis client: redis is disabled")
	}

	opt, err := goredis.ParseURL(cfg.URL)
	if err != nil {
		return nil, fmt.Errorf("parse redis URL: %w", sanitizeError(err))
	}

	if cfg.Password != "" {
		opt.Password = cfg.Password
	}
	if cfg.DB >= 0 {
		opt.DB = cfg.DB
	}
	if cfg.PoolSize > 0 {
		opt.PoolSize = cfg.PoolSize
	}
	if cfg.DialTimeout > 0 {
		opt.DialTimeout = cfg.DialTimeout
	}
	if cfg.ReadTimeout > 0 {
		opt.ReadTimeout = cfg.ReadTimeout
	}
	if cfg.WriteTimeout > 0 {
		opt.WriteTimeout = cfg.WriteTimeout
	}

	rdb := goredis.NewClient(opt)

	dialTimeout := cfg.DialTimeout
	if dialTimeout <= 0 {
		dialTimeout = 5 * time.Second
	}
	pingCtx, cancel := context.WithTimeout(ctx, dialTimeout)
	defer cancel()

	if err := rdb.Ping(pingCtx).Err(); err != nil {
		_ = rdb.Close()
		return nil, fmt.Errorf("redis ping failed: %w", sanitizeError(err))
	}

	return &Client{rdb: rdb}, nil
}

// Ping sends a PING command to verify the connection is alive.
func (c *Client) Ping(ctx context.Context) error {
	if c == nil || c.rdb == nil {
		return errors.New("redis client is not initialized")
	}
	return c.rdb.Ping(ctx).Err()
}

// Close gracefully closes the Redis client connection pool.
func (c *Client) Close() error {
	if c == nil || c.rdb == nil {
		return nil
	}
	return c.rdb.Close()
}

// RDB exposes the underlying go-redis Client for direct low-level commands.
func (c *Client) RDB() *goredis.Client {
	if c == nil {
		return nil
	}
	return c.rdb
}

// HealthCheck returns a readiness check closure for Redis.
// It executes a PING within the given timeout and returns a sanitized error on failure.
func HealthCheck(client *Client, timeout time.Duration) func(ctx context.Context) error {
	return func(ctx context.Context) error {
		if client == nil || client.rdb == nil {
			return errors.New("redis client is not initialized")
		}
		if timeout <= 0 {
			timeout = 2 * time.Second
		}
		checkCtx, cancel := context.WithTimeout(ctx, timeout)
		defer cancel()

		if err := client.Ping(checkCtx); err != nil {
			return errors.New("redis ping failed")
		}
		return nil
	}
}

// sanitizeError ensures no passwords or raw credentials are leaked in error strings.
func sanitizeError(err error) error {
	if err == nil {
		return nil
	}
	msg := err.Error()
	if strings.Contains(msg, ":") && strings.Contains(msg, "@") {
		// Replace user:password@ with ***@
		atIdx := strings.LastIndex(msg, "@")
		colonIdx := strings.LastIndex(msg[:atIdx], ":")
		if colonIdx != -1 {
			msg = msg[:colonIdx+1] + "***" + msg[atIdx:]
		}
	}
	return errors.New(msg)
}
