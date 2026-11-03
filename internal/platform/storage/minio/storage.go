package minio

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/url"
	"time"

	miniosdk "github.com/minio/minio-go/v7"
)

var (
	// ErrObjectNotFound indicates the requested object does not exist in the bucket.
	ErrObjectNotFound = errors.New("storage: object not found")

	// ErrNilClient indicates the operation was attempted on an uninitialized MinIO client.
	ErrNilClient = errors.New("storage: minio client is nil")

	// ErrEmptyKey indicates an empty object key was provided.
	ErrEmptyKey = errors.New("storage: object key must not be empty")
)

// ObjectInfo holds metadata about a stored object.
type ObjectInfo struct {
	Key          string
	Size         int64
	ContentType  string
	LastModified time.Time
	ETag         string
}

// Storage defines the contract for object storage operations.
type Storage interface {
	PutObject(ctx context.Context, key string, reader io.Reader, size int64, contentType string) error
	GetObject(ctx context.Context, key string) (io.ReadCloser, error)
	StatObject(ctx context.Context, key string) (*ObjectInfo, error)
	DeleteObject(ctx context.Context, key string) error
	PresignedGetURL(ctx context.Context, key string, expiry time.Duration) (string, error)
	PresignedPutURL(ctx context.Context, key string, expiry time.Duration) (string, error)
}

// Ensure Client implements Storage.
var _ Storage = (*Client)(nil)

// PutObject uploads an object from reader into the configured bucket.
func (c *Client) PutObject(ctx context.Context, key string, reader io.Reader, size int64, contentType string) error {
	if c == nil || c.client == nil {
		return ErrNilClient
	}
	if key == "" {
		return ErrEmptyKey
	}
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	opts := miniosdk.PutObjectOptions{
		ContentType: contentType,
	}

	_, err := c.client.PutObject(ctx, c.bucket, key, reader, size, opts)
	if err != nil {
		return fmt.Errorf("storage put object %q: %w", key, err)
	}
	return nil
}

// GetObject retrieves an object by key. The caller must close the returned io.ReadCloser.
// Returns ErrObjectNotFound if the object does not exist.
func (c *Client) GetObject(ctx context.Context, key string) (io.ReadCloser, error) {
	if c == nil || c.client == nil {
		return nil, ErrNilClient
	}
	if key == "" {
		return nil, ErrEmptyKey
	}

	obj, err := c.client.GetObject(ctx, c.bucket, key, miniosdk.GetObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("storage get object %q: %w", key, err)
	}

	// Verify the object actually exists by calling Stat()
	_, statErr := obj.Stat()
	if statErr != nil {
		_ = obj.Close()
		errResp := miniosdk.ToErrorResponse(statErr)
		if errResp.Code == "NoSuchKey" {
			return nil, ErrObjectNotFound
		}
		return nil, fmt.Errorf("storage stat object %q: %w", key, statErr)
	}

	return obj, nil
}

// StatObject retrieves metadata for an object without downloading its content.
func (c *Client) StatObject(ctx context.Context, key string) (*ObjectInfo, error) {
	if c == nil || c.client == nil {
		return nil, ErrNilClient
	}
	if key == "" {
		return nil, ErrEmptyKey
	}

	info, err := c.client.StatObject(ctx, c.bucket, key, miniosdk.StatObjectOptions{})
	if err != nil {
		errResp := miniosdk.ToErrorResponse(err)
		if errResp.Code == "NoSuchKey" {
			return nil, ErrObjectNotFound
		}
		return nil, fmt.Errorf("storage stat object %q: %w", key, err)
	}

	return &ObjectInfo{
		Key:          info.Key,
		Size:         info.Size,
		ContentType:  info.ContentType,
		LastModified: info.LastModified,
		ETag:         info.ETag,
	}, nil
}

// DeleteObject deletes an object by key from the configured bucket.
func (c *Client) DeleteObject(ctx context.Context, key string) error {
	if c == nil || c.client == nil {
		return ErrNilClient
	}
	if key == "" {
		return ErrEmptyKey
	}

	err := c.client.RemoveObject(ctx, c.bucket, key, miniosdk.RemoveObjectOptions{})
	if err != nil {
		return fmt.Errorf("storage delete object %q: %w", key, err)
	}
	return nil
}

// PresignedGetURL generates a presigned GET URL for downloading an object with an expiry duration.
func (c *Client) PresignedGetURL(ctx context.Context, key string, expiry time.Duration) (string, error) {
	if c == nil || c.client == nil {
		return "", ErrNilClient
	}
	if key == "" {
		return "", ErrEmptyKey
	}
	if expiry <= 0 {
		expiry = 15 * time.Minute
	}

	u, err := c.client.PresignedGetObject(ctx, c.bucket, key, expiry, url.Values{})
	if err != nil {
		return "", fmt.Errorf("storage presigned get %q: %w", key, err)
	}
	return u.String(), nil
}

// PresignedPutURL generates a presigned PUT URL for uploading an object directly to the bucket.
func (c *Client) PresignedPutURL(ctx context.Context, key string, expiry time.Duration) (string, error) {
	if c == nil || c.client == nil {
		return "", ErrNilClient
	}
	if key == "" {
		return "", ErrEmptyKey
	}
	if expiry <= 0 {
		expiry = 15 * time.Minute
	}

	u, err := c.client.PresignedPutObject(ctx, c.bucket, key, expiry)
	if err != nil {
		return "", fmt.Errorf("storage presigned put %q: %w", key, err)
	}
	return u.String(), nil
}
