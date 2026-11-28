// Package storage talks to Neon Object Storage over the S3 API.
package storage

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// Config holds Neon Object Storage settings from the AWS_* env vars
// that `neon deploy` / `neon env pull` write.
type Config struct {
	AccessKeyID     string
	SecretAccessKey string
	Endpoint        string
	Region          string
	Bucket          string
}

// ConfigFromEnv reads the Neon-injected AWS_* variables.
// Bucket defaults to "excel" when STORAGE_BUCKET is unset.
func ConfigFromEnv() (Config, error) {
	cfg := Config{
		AccessKeyID:     os.Getenv("AWS_ACCESS_KEY_ID"),
		SecretAccessKey: os.Getenv("AWS_SECRET_ACCESS_KEY"),
		Endpoint:        os.Getenv("AWS_ENDPOINT_URL_S3"),
		Region:          os.Getenv("AWS_REGION"),
		Bucket:          os.Getenv("STORAGE_BUCKET"),
	}
	if cfg.Bucket == "" {
		cfg.Bucket = "excel"
	}
	var missing []string
	if cfg.AccessKeyID == "" {
		missing = append(missing, "AWS_ACCESS_KEY_ID")
	}
	if cfg.SecretAccessKey == "" {
		missing = append(missing, "AWS_SECRET_ACCESS_KEY")
	}
	if cfg.Endpoint == "" {
		missing = append(missing, "AWS_ENDPOINT_URL_S3")
	}
	if cfg.Region == "" {
		missing = append(missing, "AWS_REGION")
	}
	if len(missing) > 0 {
		return Config{}, fmt.Errorf("object storage env required: %s", strings.Join(missing, ", "))
	}
	return cfg, nil
}

// Client is a path-style S3 client aimed at a Neon Object Storage endpoint.
type Client struct {
	s3     *s3.Client
	bucket string
}

// NewClient builds a Neon Object Storage client. forcePathStyle is required.
func NewClient(cfg Config) *Client {
	s3Client := s3.New(s3.Options{
		BaseEndpoint: aws.String(cfg.Endpoint),
		Region:       cfg.Region,
		Credentials:  credentials.NewStaticCredentialsProvider(cfg.AccessKeyID, cfg.SecretAccessKey, ""),
		UsePathStyle: true,
	})
	return &Client{s3: s3Client, bucket: cfg.Bucket}
}

// PutObject uploads body under key.
func (c *Client) PutObject(ctx context.Context, key string, body io.Reader, contentType string) error {
	if key == "" {
		return errors.New("object key is required")
	}
	input := &s3.PutObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(key),
		Body:   body,
	}
	if contentType != "" {
		input.ContentType = aws.String(contentType)
	}
	_, err := c.s3.PutObject(ctx, input)
	if err != nil {
		return fmt.Errorf("put object %q: %w", key, err)
	}
	return nil
}

// GetObject downloads key and returns the body bytes.
func (c *Client) GetObject(ctx context.Context, key string) ([]byte, error) {
	if key == "" {
		return nil, errors.New("object key is required")
	}
	out, err := c.s3.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, fmt.Errorf("get object %q: %w", key, err)
	}
	defer out.Body.Close()
	data, err := io.ReadAll(out.Body)
	if err != nil {
		return nil, fmt.Errorf("read object %q: %w", key, err)
	}
	return data, nil
}

// Bucket returns the configured bucket name.
func (c *Client) Bucket() string {
	return c.bucket
}
