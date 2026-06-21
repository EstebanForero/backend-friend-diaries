package storage

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"time"

	"github.com/friend-diaries/backend/internal/config"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type Client struct {
	mc        *minio.Client
	bucket    string
	publicURL string
}

func NewClient(cfg config.StorageConfig) (*Client, error) {
	mc, err := minio.New(cfg.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure: cfg.UseSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("creating minio client: %w", err)
	}

	return &Client{mc: mc, bucket: cfg.Bucket, publicURL: cfg.PublicURL}, nil
}

// EnsureBucket creates the bucket if it does not exist.
func (c *Client) EnsureBucket(ctx context.Context) error {
	exists, err := c.mc.BucketExists(ctx, c.bucket)
	if err != nil {
		return fmt.Errorf("checking bucket: %w", err)
	}
	if !exists {
		if err := c.mc.MakeBucket(ctx, c.bucket, minio.MakeBucketOptions{}); err != nil {
			return fmt.Errorf("creating bucket: %w", err)
		}
	}
	return nil
}

// Upload stores an object and returns its key.
func (c *Client) Upload(ctx context.Context, key string, r io.Reader, size int64, contentType string) error {
	_, err := c.mc.PutObject(ctx, c.bucket, key, r, size, minio.PutObjectOptions{
		ContentType: contentType,
	})
	if err != nil {
		return fmt.Errorf("uploading object: %w", err)
	}
	return nil
}

// PresignedURL returns a time-limited GET URL for the given key, rewritten to
// use the public URL so browsers can reach MinIO even when the client connects
// via an internal endpoint (e.g. minio:9000).
func (c *Client) PresignedURL(ctx context.Context, key string, expiry time.Duration) (string, error) {
	u, err := c.mc.PresignedGetObject(ctx, c.bucket, key, expiry, nil)
	if err != nil {
		return "", fmt.Errorf("generating presigned url: %w", err)
	}

	if c.publicURL != "" {
		pub, err := url.Parse(c.publicURL)
		if err == nil {
			u.Scheme = pub.Scheme
			u.Host = pub.Host
		}
	}

	return u.String(), nil
}

// Delete removes an object from the bucket.
func (c *Client) Delete(ctx context.Context, key string) error {
	return c.mc.RemoveObject(ctx, c.bucket, key, minio.RemoveObjectOptions{})
}
