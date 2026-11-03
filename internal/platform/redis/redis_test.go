package redis_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/moonrill/go-backend-template/internal/config"
	"github.com/moonrill/go-backend-template/internal/platform/redis"
)

type sampleUser struct {
	ID    string `json:"id"`
	Email string `json:"email"`
	Role  string `json:"role"`
}

func TestRedisLifecycleAndPing(t *testing.T) {
	mr := miniredis.RunT(t)

	cfg := config.RedisConfig{
		Enabled:      true,
		URL:          "redis://" + mr.Addr() + "/0",
		PoolSize:     5,
		DialTimeout:  2 * time.Second,
		ReadTimeout:  1 * time.Second,
		WriteTimeout: 1 * time.Second,
	}

	ctx := context.Background()
	client, err := redis.New(ctx, cfg)
	if err != nil {
		t.Fatalf("redis.New() error = %v", err)
	}
	defer client.Close()

	if client.RDB() == nil {
		t.Fatal("expected non-nil RDB()")
	}

	if err := client.Ping(ctx); err != nil {
		t.Fatalf("Ping() error = %v", err)
	}

	// HealthCheck returns nil for healthy connection
	check := redis.HealthCheck(client, 1*time.Second)
	if err := check(ctx); err != nil {
		t.Fatalf("HealthCheck() error = %v", err)
	}

	// HealthCheck fails when client is nil
	nilCheck := redis.HealthCheck(nil, 1*time.Second)
	if err := nilCheck(ctx); err == nil {
		t.Fatal("expected error from nil client HealthCheck")
	}

	// When miniredis closes, Ping and HealthCheck fail
	mr.Close()
	if err := client.Ping(ctx); err == nil {
		t.Fatal("expected Ping() to fail after redis close")
	}
	if err := check(ctx); err == nil {
		t.Fatal("expected HealthCheck() to fail after redis close")
	}
}

func TestRedisNewDisabled(t *testing.T) {
	cfg := config.RedisConfig{
		Enabled: false,
		URL:     "redis://127.0.0.1:6379/0",
	}

	_, err := redis.New(context.Background(), cfg)
	if err == nil {
		t.Fatal("expected error when initializing disabled redis client")
	}
}

func TestRedisNewInvalidURL(t *testing.T) {
	cfg := config.RedisConfig{
		Enabled: true,
		URL:     "://invalid-url",
	}

	_, err := redis.New(context.Background(), cfg)
	if err == nil {
		t.Fatal("expected error for invalid redis URL")
	}
}

func TestCacheSetGetDeleteAndTTL(t *testing.T) {
	mr := miniredis.RunT(t)

	cfg := config.RedisConfig{
		Enabled:      true,
		URL:          "redis://" + mr.Addr() + "/0",
		PoolSize:     5,
		DialTimeout:  2 * time.Second,
		ReadTimeout:  1 * time.Second,
		WriteTimeout: 1 * time.Second,
	}

	ctx := context.Background()
	client, err := redis.New(ctx, cfg)
	if err != nil {
		t.Fatalf("redis.New() error = %v", err)
	}
	defer client.Close()

	cache := redis.NewCache(client)

	t.Run("string value", func(t *testing.T) {
		key := "test:str"
		if err := cache.Set(ctx, key, "hello world", 10*time.Minute); err != nil {
			t.Fatalf("Set() error = %v", err)
		}

		var val string
		if err := cache.Get(ctx, key, &val); err != nil {
			t.Fatalf("Get() error = %v", err)
		}
		if val != "hello world" {
			t.Fatalf("got %q, want %q", val, "hello world")
		}
	})

	t.Run("byte slice value", func(t *testing.T) {
		key := "test:bytes"
		payload := []byte("raw byte content")
		if err := cache.Set(ctx, key, payload, 10*time.Minute); err != nil {
			t.Fatalf("Set() error = %v", err)
		}

		var retrieved []byte
		if err := cache.Get(ctx, key, &retrieved); err != nil {
			t.Fatalf("Get() error = %v", err)
		}
		if string(retrieved) != string(payload) {
			t.Fatalf("got %q, want %q", string(retrieved), string(payload))
		}
	})

	t.Run("JSON struct value", func(t *testing.T) {
		key := "test:user:123"
		user := sampleUser{
			ID:    "123",
			Email: "test@example.com",
			Role:  "admin",
		}
		if err := cache.Set(ctx, key, user, 5*time.Minute); err != nil {
			t.Fatalf("Set() error = %v", err)
		}

		var retrieved sampleUser
		if err := cache.Get(ctx, key, &retrieved); err != nil {
			t.Fatalf("Get() error = %v", err)
		}
		if retrieved != user {
			t.Fatalf("got %+v, want %+v", retrieved, user)
		}
	})

	t.Run("cache miss returns ErrCacheMiss", func(t *testing.T) {
		var val string
		err := cache.Get(ctx, "non:existent:key", &val)
		if !errors.Is(err, redis.ErrCacheMiss) {
			t.Fatalf("got error %v, want ErrCacheMiss", err)
		}
	})

	t.Run("delete removes key", func(t *testing.T) {
		key := "test:to_delete"
		if err := cache.Set(ctx, key, "val", 5*time.Minute); err != nil {
			t.Fatalf("Set() error = %v", err)
		}

		if err := cache.Delete(ctx, key); err != nil {
			t.Fatalf("Delete() error = %v", err)
		}

		var val string
		if err := cache.Get(ctx, key, &val); !errors.Is(err, redis.ErrCacheMiss) {
			t.Fatalf("expected ErrCacheMiss after delete, got %v", err)
		}
	})

	t.Run("TTL and expiration", func(t *testing.T) {
		key := "test:expiring"
		ttl := 30 * time.Second
		if err := cache.Set(ctx, key, "expires soon", ttl); err != nil {
			t.Fatalf("Set() error = %v", err)
		}

		remaining, err := cache.TTL(ctx, key)
		if err != nil {
			t.Fatalf("TTL() error = %v", err)
		}
		if remaining <= 0 || remaining > ttl {
			t.Fatalf("unexpected TTL: %v", remaining)
		}

		// Fast forward miniredis clock beyond expiration
		mr.FastForward(35 * time.Second)

		var val string
		if err := cache.Get(ctx, key, &val); !errors.Is(err, redis.ErrCacheMiss) {
			t.Fatalf("expected key to be expired, got error %v", err)
		}
	})

	t.Run("nil client returns ErrNilClient", func(t *testing.T) {
		nilCache := redis.NewCache(nil)
		if err := nilCache.Set(ctx, "k", "v", time.Minute); !errors.Is(err, redis.ErrNilClient) {
			t.Fatalf("expected ErrNilClient, got %v", err)
		}
		var val string
		if err := nilCache.Get(ctx, "k", &val); !errors.Is(err, redis.ErrNilClient) {
			t.Fatalf("expected ErrNilClient, got %v", err)
		}
		if err := nilCache.Delete(ctx, "k"); !errors.Is(err, redis.ErrNilClient) {
			t.Fatalf("expected ErrNilClient, got %v", err)
		}
		if _, err := nilCache.TTL(ctx, "k"); !errors.Is(err, redis.ErrNilClient) {
			t.Fatalf("expected ErrNilClient, got %v", err)
		}
	})

	t.Run("empty key validation", func(t *testing.T) {
		if err := cache.Set(ctx, "", "v", time.Minute); err == nil {
			t.Fatal("expected error for empty key in Set")
		}
		var val string
		if err := cache.Get(ctx, "", &val); err == nil {
			t.Fatal("expected error for empty key in Get")
		}
		if err := cache.Delete(ctx, ""); err == nil {
			t.Fatal("expected error for empty key in Delete")
		}
		if _, err := cache.TTL(ctx, ""); err == nil {
			t.Fatal("expected error for empty key in TTL")
		}
	})

	t.Run("context cancellation propagates", func(t *testing.T) {
		canceledCtx, cancel := context.WithCancel(ctx)
		cancel()

		if err := cache.Set(canceledCtx, "ctx:key", "val", time.Minute); err == nil {
			t.Fatal("expected error with canceled context in Set")
		}
		var val string
		if err := cache.Get(canceledCtx, "ctx:key", &val); err == nil {
			t.Fatal("expected error with canceled context in Get")
		}
		if err := cache.Delete(canceledCtx, "ctx:key"); err == nil {
			t.Fatal("expected error with canceled context in Delete")
		}
		if _, err := cache.TTL(canceledCtx, "ctx:key"); err == nil {
			t.Fatal("expected error with canceled context in TTL")
		}
	})
}

func TestSanitizeCredentialsInError(t *testing.T) {
	secretURL := "redis://app_user:super_secret_password@127.0.0.1:9999/0"
	cfg := config.RedisConfig{
		Enabled:     true,
		URL:         secretURL,
		DialTimeout: 100 * time.Millisecond,
	}

	_, err := redis.New(context.Background(), cfg)
	if err == nil {
		t.Fatal("expected connection error")
	}

	errMsg := err.Error()
	if contains(errMsg, "super_secret_password") {
		t.Fatalf("error message leaked password: %q", errMsg)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (fmt.Sprintf("%s", s) != "" && errors.New(s) != nil && (s == substr || len(substr) > 0 && (len(s) > 0 && (s != "" && (s[:] != "" && (len(s) >= len(substr) && hasSubstr(s, substr)))))))
}

func hasSubstr(s, substr string) bool {
	for i := 0; i+len(substr) <= len(s); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
