package minio

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	miniosdk "github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/moonrill/go-backend-template/internal/config"
)

// Client wraps the MinIO SDK client with connection lifecycle, health check, and storage helpers.
type Client struct {
	client *miniosdk.Client
	bucket string
}

// New creates and validates a new MinIO connection client based on the provided configuration.
func New(ctx context.Context, cfg config.MinIOConfig) (*Client, error) {
	if !cfg.Enabled {
		return nil, errors.New("cannot initialize minio client: minio is disabled")
	}

	creds := credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, "")
	client, err := miniosdk.New(cfg.Endpoint, &miniosdk.Options{
		Creds:  creds,
		Secure: cfg.UseSSL,
		Region: cfg.Region,
	})
	if err != nil {
		return nil, fmt.Errorf("minio client creation failed: %w", sanitizeError(err))
	}

	connectTimeout := cfg.ConnectTimeout
	if connectTimeout <= 0 {
		connectTimeout = 5 * time.Second
	}
	pingCtx, cancel := context.WithTimeout(ctx, connectTimeout)
	defer cancel()

	exists, err := client.BucketExists(pingCtx, cfg.Bucket)
	if err != nil {
		return nil, fmt.Errorf("minio bucket check failed: %w", sanitizeError(err))
	}

	if !exists {
		if cfg.AutoCreate {
			makeErr := client.MakeBucket(pingCtx, cfg.Bucket, miniosdk.MakeBucketOptions{Region: cfg.Region})
			if makeErr != nil {
				return nil, fmt.Errorf("minio create bucket %q failed: %w", cfg.Bucket, sanitizeError(makeErr))
			}
		} else {
			return nil, fmt.Errorf("minio bucket %q does not exist", cfg.Bucket)
		}
	}

	return &Client{
		client: client,
		bucket: cfg.Bucket,
	}, nil
}

// Bucket returns the default bucket name associated with this client.
func (c *Client) Bucket() string {
	if c == nil {
		return ""
	}
	return c.bucket
}

// SDKClient exposes the underlying MinIO Go SDK Client for low-level or advanced operations.
func (c *Client) SDKClient() *miniosdk.Client {
	if c == nil {
		return nil
	}
	return c.client
}

// Ping verifies that MinIO is reachable and that the configured bucket exists.
func (c *Client) Ping(ctx context.Context) error {
	if c == nil || c.client == nil {
		return errors.New("minio client is not initialized")
	}
	exists, err := c.client.BucketExists(ctx, c.bucket)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("bucket %q does not exist", c.bucket)
	}
	return nil
}

// Close closes any idle HTTP transport connections used by the MinIO client.
func (c *Client) Close() error {
	// minio-go manages HTTP transports internally. This method provides interface symmetry.
	return nil
}

// HealthCheck returns a readiness check closure for MinIO.
// It checks bucket accessibility within the given timeout and returns a sanitized error on failure.
func HealthCheck(client *Client, timeout time.Duration) func(ctx context.Context) error {
	return func(ctx context.Context) error {
		if client == nil || client.client == nil {
			return errors.New("minio client is not initialized")
		}
		if timeout <= 0 {
			timeout = 2 * time.Second
		}
		checkCtx, cancel := context.WithTimeout(ctx, timeout)
		defer cancel()

		if err := client.Ping(checkCtx); err != nil {
			return errors.New("minio ping failed")
		}
		return nil
	}
}

// sanitizeError ensures no secret credentials or access keys are leaked in error strings.
func sanitizeError(err error) error {
	if err == nil {
		return nil
	}
	msg := err.Error()
	if strings.Contains(msg, ":") && strings.Contains(msg, "@") {
		atIdx := strings.LastIndex(msg, "@")
		colonIdx := strings.LastIndex(msg[:atIdx], ":")
		if colonIdx != -1 {
			msg = msg[:colonIdx+1] + "***" + msg[atIdx:]
		}
	}
	return errors.New(msg)
}
