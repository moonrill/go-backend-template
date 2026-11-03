package minio_test

import (
	"bytes"
	"context"
	"crypto/md5"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/moonrill/go-backend-template/internal/config"
	"github.com/moonrill/go-backend-template/internal/platform/storage/minio"
)

func TestMinIONewDisabled(t *testing.T) {
	cfg := config.MinIOConfig{
		Enabled: false,
	}

	_, err := minio.New(context.Background(), cfg)
	if err == nil {
		t.Fatal("expected error when initializing disabled minio client")
	}
}

func TestMinIONilClientSafety(t *testing.T) {
	var c *minio.Client
	ctx := context.Background()

	if err := c.Ping(ctx); err == nil {
		t.Fatal("expected error from nil client Ping")
	}

	hc := minio.HealthCheck(c, time.Second)
	if err := hc(ctx); err == nil {
		t.Fatal("expected error from nil client HealthCheck")
	}

	if err := c.PutObject(ctx, "k", strings.NewReader("v"), 1, "text/plain"); !errors.Is(err, minio.ErrNilClient) {
		t.Fatalf("expected ErrNilClient from PutObject, got %v", err)
	}

	if _, err := c.GetObject(ctx, "k"); !errors.Is(err, minio.ErrNilClient) {
		t.Fatalf("expected ErrNilClient from GetObject, got %v", err)
	}

	if _, err := c.StatObject(ctx, "k"); !errors.Is(err, minio.ErrNilClient) {
		t.Fatalf("expected ErrNilClient from StatObject, got %v", err)
	}

	if err := c.DeleteObject(ctx, "k"); !errors.Is(err, minio.ErrNilClient) {
		t.Fatalf("expected ErrNilClient from DeleteObject, got %v", err)
	}

	if _, err := c.PresignedGetURL(ctx, "k", time.Minute); !errors.Is(err, minio.ErrNilClient) {
		t.Fatalf("expected ErrNilClient from PresignedGetURL, got %v", err)
	}

	if _, err := c.PresignedPutURL(ctx, "k", time.Minute); !errors.Is(err, minio.ErrNilClient) {
		t.Fatalf("expected ErrNilClient from PresignedPutURL, got %v", err)
	}

	if c.Bucket() != "" {
		t.Fatalf("expected empty string from nil client Bucket(), got %q", c.Bucket())
	}

	if c.SDKClient() != nil {
		t.Fatal("expected nil from nil client SDKClient()")
	}

	if err := c.Close(); err != nil {
		t.Fatalf("expected nil from nil client Close(), got %v", err)
	}
}

func TestMinIOMockServerLifecycleAndStorage(t *testing.T) {
	storage := make(map[string][]byte)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/")
		parts := strings.SplitN(path, "/", 2)
		bucket := parts[0]
		key := ""
		if len(parts) > 1 {
			key = parts[1]
		}

		switch r.Method {
		case http.MethodHead:
			if key == "" {
				// BucketExists check
				if bucket == "my-bucket" {
					w.WriteHeader(http.StatusOK)
					return
				}
				w.WriteHeader(http.StatusNotFound)
				return
			}
			// StatObject check
			data, ok := storage[key]
			if !ok {
				w.WriteHeader(http.StatusNotFound)
				return
			}
			etag := fmt.Sprintf("\"%x\"", md5.Sum(data))
			w.Header().Set("Content-Length", strconv.Itoa(len(data)))
			w.Header().Set("Content-Type", "text/plain")
			w.Header().Set("ETag", etag)
			w.Header().Set("Last-Modified", time.Now().UTC().Format(http.TimeFormat))
			w.WriteHeader(http.StatusOK)

		case http.MethodGet:
			if key == "" {
				// Location or bucket query
				w.Header().Set("Content-Type", "application/xml")
				_, _ = w.Write([]byte(`<LocationConstraint xmlns="http://s3.amazonaws.com/doc/2006-03-01/">us-east-1</LocationConstraint>`))
				return
			}
			data, ok := storage[key]
			if !ok {
				w.Header().Set("Content-Type", "application/xml")
				w.WriteHeader(http.StatusNotFound)
				_, _ = w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?><Error><Code>NoSuchKey</Code><Message>The specified key does not exist.</Message></Error>`))
				return
			}
			etag := fmt.Sprintf("\"%x\"", md5.Sum(data))
			w.Header().Set("Content-Length", strconv.Itoa(len(data)))
			w.Header().Set("Content-Type", "text/plain")
			w.Header().Set("ETag", etag)
			w.Header().Set("Last-Modified", time.Now().UTC().Format(http.TimeFormat))
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(data)

		case http.MethodPut:
			if key == "" {
				// MakeBucket
				w.WriteHeader(http.StatusOK)
				return
			}
			body, _ := io.ReadAll(r.Body)
			data := extractChunkedPayload(body)
			storage[key] = data
			etag := fmt.Sprintf("\"%x\"", md5.Sum(data))
			w.Header().Set("ETag", etag)
			w.WriteHeader(http.StatusOK)

		case http.MethodDelete:
			delete(storage, key)
			w.WriteHeader(http.StatusNoContent)

		default:
			w.WriteHeader(http.StatusBadRequest)
		}
	}))
	defer ts.Close()

	endpoint := strings.TrimPrefix(ts.URL, "http://")

	cfg := config.MinIOConfig{
		Enabled:        true,
		Endpoint:       endpoint,
		Region:         "us-east-1",
		Bucket:         "my-bucket",
		AccessKey:      "test-access-key",
		SecretKey:      "test-secret-key",
		UseSSL:         false,
		ConnectTimeout: 2 * time.Second,
		AutoCreate:     true,
	}

	ctx := context.Background()
	client, err := minio.New(ctx, cfg)
	if err != nil {
		t.Fatalf("minio.New() error = %v", err)
	}
	defer client.Close()

	if client.Bucket() != "my-bucket" {
		t.Fatalf("Bucket() = %q, want my-bucket", client.Bucket())
	}
	if client.SDKClient() == nil {
		t.Fatal("expected non-nil SDKClient()")
	}

	// Ping & HealthCheck
	if err := client.Ping(ctx); err != nil {
		t.Fatalf("Ping() error = %v", err)
	}

	hc := minio.HealthCheck(client, time.Second)
	if err := hc(ctx); err != nil {
		t.Fatalf("HealthCheck() error = %v", err)
	}

	// PutObject
	testContent := "hello minio object storage"
	err = client.PutObject(ctx, "test/doc.txt", strings.NewReader(testContent), int64(len(testContent)), "text/plain")
	if err != nil {
		t.Fatalf("PutObject() error = %v", err)
	}

	// StatObject
	info, err := client.StatObject(ctx, "test/doc.txt")
	if err != nil {
		t.Fatalf("StatObject() error = %v", err)
	}
	if info.Key != "test/doc.txt" {
		t.Fatalf("info.Key = %q, want test/doc.txt", info.Key)
	}

	// GetObject
	rc, err := client.GetObject(ctx, "test/doc.txt")
	if err != nil {
		t.Fatalf("GetObject() error = %v", err)
	}
	buf := new(bytes.Buffer)
	_, _ = buf.ReadFrom(rc)
	_ = rc.Close()
	if buf.String() != testContent {
		t.Fatalf("got %q, want %q", buf.String(), testContent)
	}

	// Presigned URLs
	getURL, err := client.PresignedGetURL(ctx, "test/doc.txt", 10*time.Minute)
	if err != nil {
		t.Fatalf("PresignedGetURL() error = %v", err)
	}
	if getURL == "" || !strings.Contains(getURL, "test/doc.txt") {
		t.Fatalf("unexpected PresignedGetURL: %s", getURL)
	}

	putURL, err := client.PresignedPutURL(ctx, "test/upload.txt", 10*time.Minute)
	if err != nil {
		t.Fatalf("PresignedPutURL() error = %v", err)
	}
	if putURL == "" || !strings.Contains(putURL, "test/upload.txt") {
		t.Fatalf("unexpected PresignedPutURL: %s", putURL)
	}

	// DeleteObject
	if err := client.DeleteObject(ctx, "test/doc.txt"); err != nil {
		t.Fatalf("DeleteObject() error = %v", err)
	}

	// GetObject on deleted key returns ErrObjectNotFound
	_, err = client.GetObject(ctx, "test/doc.txt")
	if !errors.Is(err, minio.ErrObjectNotFound) {
		t.Fatalf("expected ErrObjectNotFound, got %v", err)
	}

	// Empty key validation
	if err := client.PutObject(ctx, "", strings.NewReader(""), 0, ""); !errors.Is(err, minio.ErrEmptyKey) {
		t.Fatalf("expected ErrEmptyKey from PutObject, got %v", err)
	}
	if _, err := client.GetObject(ctx, ""); !errors.Is(err, minio.ErrEmptyKey) {
		t.Fatalf("expected ErrEmptyKey from GetObject, got %v", err)
	}
	if _, err := client.StatObject(ctx, ""); !errors.Is(err, minio.ErrEmptyKey) {
		t.Fatalf("expected ErrEmptyKey from StatObject, got %v", err)
	}
	if err := client.DeleteObject(ctx, ""); !errors.Is(err, minio.ErrEmptyKey) {
		t.Fatalf("expected ErrEmptyKey from DeleteObject, got %v", err)
	}
	if _, err := client.PresignedGetURL(ctx, "", time.Minute); !errors.Is(err, minio.ErrEmptyKey) {
		t.Fatalf("expected ErrEmptyKey from PresignedGetURL, got %v", err)
	}
	if _, err := client.PresignedPutURL(ctx, "", time.Minute); !errors.Is(err, minio.ErrEmptyKey) {
		t.Fatalf("expected ErrEmptyKey from PresignedPutURL, got %v", err)
	}
}

func TestMinIOSanitizeCredentialsInError(t *testing.T) {
	cfg := config.MinIOConfig{
		Enabled:        true,
		Endpoint:       "127.0.0.1:9999",
		Region:         "us-east-1",
		Bucket:         "secret-bucket",
		AccessKey:      "super_secret_access_key",
		SecretKey:      "super_secret_secret_key",
		UseSSL:         false,
		ConnectTimeout: 100 * time.Millisecond,
	}

	_, err := minio.New(context.Background(), cfg)
	if err == nil {
		t.Fatal("expected connection error")
	}

	errMsg := err.Error()
	if strings.Contains(errMsg, "super_secret_access_key") || strings.Contains(errMsg, "super_secret_secret_key") {
		t.Fatalf("error message leaked secret credentials: %q", errMsg)
	}
}

func extractChunkedPayload(body []byte) []byte {
	str := string(body)
	if !strings.Contains(str, "chunk-signature=") {
		return body
	}
	parts := strings.Split(str, "\r\n")
	var result []byte
	for i := 0; i < len(parts); i++ {
		line := parts[i]
		if strings.Contains(line, "chunk-signature=") {
			if strings.HasPrefix(line, "0;") {
				break
			}
			if i+1 < len(parts) {
				result = append(result, []byte(parts[i+1])...)
				i++
			}
		}
	}
	return result
}
