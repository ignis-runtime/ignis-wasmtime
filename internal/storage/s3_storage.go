package storage

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

// S3Storage defines the interface for S3-compatible storage operations
type S3Storage interface {
	UploadFile(ctx context.Context, key string, fileData []byte) error
	DownloadFile(ctx context.Context, key string) ([]byte, error)
	DeleteFile(ctx context.Context, key string) error
	FileExists(ctx context.Context, key string) (bool, error)
}

type s3Storage struct {
	client *s3.Client
	bucket string
}

type S3Config struct {
	Endpoint        string
	AccessKeyID     string
	SecretAccessKey string
	BucketName      string
	Region          string
}

func NewS3Storage(cfg S3Config) (S3Storage, error) {
	// 1. Load the base configuration
	// We use config.LoadDefaultConfig but override credentials and endpoint
	awsCfg, err := config.LoadDefaultConfig(context.Background(),
		config.WithRegion(cfg.Region),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			cfg.AccessKeyID,
			cfg.SecretAccessKey,
			"",
		)),
	)
	if err != nil {
		return nil, fmt.Errorf("unable to load SDK config: %w", err)
	}

	// 2. Create S3 client with BaseEndpoint (the modern way to handle custom URLs)
	client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		// UsePathStyle is required for R2 and most local S3-compat layers
		o.UsePathStyle = true
		if cfg.Endpoint != "" {
			o.BaseEndpoint = aws.String(cfg.Endpoint)
		}
	})

	return &s3Storage{
		client: client,
		bucket: cfg.BucketName,
	}, nil
}

func (s *s3Storage) UploadFile(ctx context.Context, key string, fileData []byte) error {
	_, err := s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
		Body:   bytes.NewReader(fileData),
	})
	if err != nil {
		return fmt.Errorf("upload failed: %w", err)
	}

	return nil
}

func (s *s3Storage) DownloadFile(ctx context.Context, key string) ([]byte, error) {
	result, err := s.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		var nsk *types.NoSuchKey
		if errors.As(err, &nsk) {
			return nil, fmt.Errorf("file not found: %s", key)
		}
		return nil, fmt.Errorf("download failed: %w", err)
	}
	defer result.Body.Close()

	return io.ReadAll(result.Body)
}

func (s *s3Storage) DeleteFile(ctx context.Context, key string) error {
	_, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return fmt.Errorf("delete failed: %w", err)
	}

	return nil
}

func (s *s3Storage) FileExists(ctx context.Context, key string) (bool, error) {
	_, err := s.client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		// S3 HeadObject returns 404 (NotFound) if it doesn't exist
		var nfe *types.NotFound
		if errors.As(err, &nfe) {
			return false, nil
		}
		return false, fmt.Errorf("existence check failed: %w", err)
	}

	return true, nil
}
