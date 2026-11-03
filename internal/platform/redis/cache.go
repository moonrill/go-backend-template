package redis

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	goredis "github.com/redis/go-redis/v9"
)

var (
	// ErrCacheMiss indicates that the requested key does not exist in Redis.
	ErrCacheMiss = errors.New("cache: key not found")

	// ErrNilClient indicates that an operation was attempted on an uninitialized Redis client.
	ErrNilClient = errors.New("cache: redis client is nil")
)

// Cache provides a lightweight, JSON-aware caching abstraction on top of Redis.
type Cache struct {
	client *Client
}

// NewCache constructs a Cache backed by the given Redis client.
func NewCache(client *Client) *Cache {
	return &Cache{client: client}
}

// Set stores a value with an optional TTL. If value is already a string or []byte,
// it is stored directly; otherwise, it is marshaled to JSON.
func (c *Cache) Set(ctx context.Context, key string, value any, ttl time.Duration) error {
	if c == nil || c.client == nil || c.client.rdb == nil {
		return ErrNilClient
	}
	if key == "" {
		return errors.New("cache: key must not be empty")
	}

	var payload any
	switch v := value.(type) {
	case string:
		payload = v
	case []byte:
		payload = v
	default:
		data, err := json.Marshal(v)
		if err != nil {
			return fmt.Errorf("cache marshal JSON: %w", err)
		}
		payload = data
	}

	if err := c.client.rdb.Set(ctx, key, payload, ttl).Err(); err != nil {
		return fmt.Errorf("cache set key %q: %w", key, err)
	}
	return nil
}

// Get retrieves a value by key and assigns or unmarshals it into dest.
// If dest is *string or *[]byte, the raw value is assigned.
// Otherwise, it is deserialized from JSON into dest.
// Returns ErrCacheMiss if the key does not exist.
func (c *Cache) Get(ctx context.Context, key string, dest any) error {
	if c == nil || c.client == nil || c.client.rdb == nil {
		return ErrNilClient
	}
	if key == "" {
		return errors.New("cache: key must not be empty")
	}

	val, err := c.client.rdb.Get(ctx, key).Result()
	if err != nil {
		if errors.Is(err, goredis.Nil) {
			return ErrCacheMiss
		}
		return fmt.Errorf("cache get key %q: %w", key, err)
	}

	switch d := dest.(type) {
	case *string:
		*d = val
		return nil
	case *[]byte:
		*d = []byte(val)
		return nil
	default:
		if err := json.Unmarshal([]byte(val), dest); err != nil {
			return fmt.Errorf("cache unmarshal JSON for key %q: %w", key, err)
		}
		return nil
	}
}

// Delete removes one or more keys from Redis.
func (c *Cache) Delete(ctx context.Context, keys ...string) error {
	if c == nil || c.client == nil || c.client.rdb == nil {
		return ErrNilClient
	}
	if len(keys) == 0 {
		return nil
	}
	for _, k := range keys {
		if k == "" {
			return errors.New("cache: key must not be empty")
		}
	}

	if err := c.client.rdb.Del(ctx, keys...).Err(); err != nil {
		return fmt.Errorf("cache delete keys: %w", err)
	}
	return nil
}

// TTL returns the remaining time to live for the given key.
func (c *Cache) TTL(ctx context.Context, key string) (time.Duration, error) {
	if c == nil || c.client == nil || c.client.rdb == nil {
		return 0, ErrNilClient
	}
	if key == "" {
		return 0, errors.New("cache: key must not be empty")
	}
	dur, err := c.client.rdb.TTL(ctx, key).Result()
	if err != nil {
		return 0, fmt.Errorf("cache ttl key %q: %w", key, err)
	}
	return dur, nil
}
