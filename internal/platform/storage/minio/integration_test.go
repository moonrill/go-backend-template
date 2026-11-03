package minio_test

import (
	"bytes"
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/moonrill/go-backend-template/internal/config"
	"github.com/moonrill/go-backend-template/internal/platform/storage/minio"
)

func TestMinIOIntegration(t *testing.T) {
	endpoint := os.Getenv("TEST_MINIO_ENDPOINT")
	if endpoint == "" {
		t.Skip("skipping MinIO integration test: TEST_MINIO_ENDPOINT is not set")
	}

	bucket := os.Getenv("TEST_MINIO_BUCKET")
	if bucket == "" {
		bucket = "test-integration-bucket"
	}
	accessKey := os.Getenv("TEST_MINIO_ACCESS_KEY")
	if accessKey == "" {
		accessKey = "minioadmin"
	}
	secretKey := os.Getenv("TEST_MINIO_SECRET_KEY")
	if secretKey == "" {
		secretKey = "minioadmin"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cfg := config.MinIOConfig{
		Enabled:        true,
		Endpoint:       endpoint,
		Region:         "us-east-1",
		Bucket:         bucket,
		AccessKey:      accessKey,
		SecretKey:      secretKey,
		UseSSL:         false,
		ConnectTimeout: 3 * time.Second,
		AutoCreate:     true,
	}

	client, err := minio.New(ctx, cfg)
	if err != nil {
		t.Fatalf("minio.New() error = %v", err)
	}
	defer client.Close()

	if err := client.Ping(ctx); err != nil {
		t.Fatalf("Ping() error = %v", err)
	}

	check := minio.HealthCheck(client, 2*time.Second)
	if err := check(ctx); err != nil {
		t.Fatalf("HealthCheck() error = %v", err)
	}

	testKey := "integration/test.txt"
	testPayload := "integration test payload content"

	if err := client.PutObject(ctx, testKey, strings.NewReader(testPayload), int64(len(testPayload)), "text/plain"); err != nil {
		t.Fatalf("PutObject() error = %v", err)
	}
	t.Cleanup(func() {
		_ = client.DeleteObject(context.Background(), testKey)
	})

	rc, err := client.GetObject(ctx, testKey)
	if err != nil {
		t.Fatalf("GetObject() error = %v", err)
	}
	buf := new(bytes.Buffer)
	_, _ = buf.ReadFrom(rc)
	_ = rc.Close()

	if buf.String() != testPayload {
		t.Fatalf("got %q, want %q", buf.String(), testPayload)
	}
}
