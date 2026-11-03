package redis_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/moonrill/go-backend-template/internal/config"
	"github.com/moonrill/go-backend-template/internal/platform/redis"
)

func TestRedisIntegration(t *testing.T) {
	redisURL := os.Getenv("TEST_REDIS_URL")
	if redisURL == "" {
		t.Skip("skipping Redis integration test: TEST_REDIS_URL is not set")
	}

	runRedisIntegrationTest(t, redisURL)
}

func TestRedisIntegrationWithTemporaryServer(t *testing.T) {
	mr := miniredis.RunT(t)
	runRedisIntegrationTest(t, "redis://"+mr.Addr()+"/0")
}

func runRedisIntegrationTest(t *testing.T, redisURL string) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cfg := config.RedisConfig{
		Enabled:      true,
		URL:          redisURL,
		PoolSize:     5,
		DialTimeout:  3 * time.Second,
		ReadTimeout:  2 * time.Second,
		WriteTimeout: 2 * time.Second,
	}

	client, err := redis.New(ctx, cfg)
	if err != nil {
		t.Fatalf("redis.New() error = %v", err)
	}
	defer client.Close()

	if err := client.Ping(ctx); err != nil {
		t.Fatalf("Ping() error = %v", err)
	}

	check := redis.HealthCheck(client, 2*time.Second)
	if err := check(ctx); err != nil {
		t.Fatalf("HealthCheck() error = %v", err)
	}

	cache := redis.NewCache(client)
	testKey := "integration:test:ping"
	if err := cache.Set(ctx, testKey, "pong", 1*time.Minute); err != nil {
		t.Fatalf("cache.Set() error = %v", err)
	}
	t.Cleanup(func() {
		_ = cache.Delete(context.Background(), testKey)
	})

	var val string
	if err := cache.Get(ctx, testKey, &val); err != nil {
		t.Fatalf("cache.Get() error = %v", err)
	}
	if val != "pong" {
		t.Fatalf("got %q, want %q", val, "pong")
	}
}
