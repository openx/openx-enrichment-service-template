package storage

import (
	"context"
	"io"

	"github.com/openx/openx-enrichment-service-template/pkg/config"

	"cloud.google.com/go/storage"
	"go.uber.org/zap"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
)

// Client represents a GCS client
type Client struct {
	client      *storage.Client
	inboxBucket *storage.BucketHandle
	logger      *zap.Logger
}

// NewClient creates a new GCS client
func NewClient(ctx context.Context, cfg *config.Config, logger *zap.Logger) (*Client, error) {
	var opts []option.ClientOption

	// If credentials path is provided, use it. Otherwise, use default credentials
	if cfg.GCSCredentials != "default" {
		opts = append(opts, option.WithCredentialsFile(cfg.GCSCredentials))
	}

	client, err := storage.NewClient(ctx, opts...)
	if err != nil {
		return nil, err
	}

	return &Client{
		client:      client,
		inboxBucket: client.Bucket(cfg.GCSInboxBucket),
		logger:      logger,
	}, nil
}

// InboxBucket returns the inbox bucket handle
func (c *Client) InboxBucket() *storage.BucketHandle {
	return c.inboxBucket
}

// ReadFile reads a file from the inbox bucket
func (c *Client) ReadFile(ctx context.Context, objectName string) ([]byte, error) {
	reader, err := c.inboxBucket.Object(objectName).NewReader(ctx)
	if err != nil {
		return nil, err
	}
	defer reader.Close()

	return io.ReadAll(reader)
}

// ListFiles lists files in the inbox bucket
func (c *Client) ListFiles(ctx context.Context, prefix string) ([]string, error) {
	var files []string
	it := c.inboxBucket.Objects(ctx, &storage.Query{
		Prefix: prefix,
	})

	for {
		attrs, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		files = append(files, attrs.Name)
	}

	return files, nil
}

// Close closes the GCS client
func (c *Client) Close() error {
	return c.client.Close()
}
