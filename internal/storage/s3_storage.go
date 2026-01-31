package storage

import (
	"bytes"
	"context"
	"fmt"
	"io"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

// S3Storage defines the interface for S3-compatible storage operations
type S3Storage interface {
	UploadFile(ctx context.Context, bucket, key string, fileData []byte) error
	DownloadFile(ctx context.Context, bucket, key string) ([]byte, error)
	DeleteFile(ctx context.Context, bucket, key string) error
	FileExists(ctx context.Context, bucket, key string) (bool, error)
}

// s3Storage implements the S3Storage interface
type s3Storage struct {
	client *s3.Client
	bucket string
}

// S3Config holds the configuration for S3 storage
type S3Config struct {
	Endpoint        string
	AccessKeyID     string
	SecretAccessKey string
	BucketName      string
	Region          string
}

// NewS3Storage creates a new instance of S3Storage
func NewS3Storage(config S3Config) (S3Storage, error) {
	// Create custom resolver for S3-compatible services like Cloudflare R2
	customResolver := aws.EndpointResolverFunc(func(service, region string) (aws.Endpoint, error) {
		return aws.Endpoint{
			URL: config.Endpoint,
		}, nil
	})

	// Create AWS configuration with custom credentials and endpoint
	awsConfig := aws.Config{
		Region: config.Region,
		Credentials: credentials.NewStaticCredentialsProvider(
			config.AccessKeyID,
			config.SecretAccessKey,
			"", // Session token not needed for R2
		),
		EndpointResolver: customResolver,
	}

	// Create S3 client
	client := s3.NewFromConfig(awsConfig, func(o *s3.Options) {
		o.UsePathStyle = true // Required for many S3-compatible services
	})

	return &s3Storage{
		client: client,
		bucket: config.BucketName,
	}, nil
}

// UploadFile uploads a file to S3-compatible storage
func (s *s3Storage) UploadFile(ctx context.Context, bucket, key string, fileData []byte) error {
	if bucket == "" {
		bucket = s.bucket
	}

	_, err := s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
		Body:   bytes.NewReader(fileData),
	})
	if err != nil {
		return fmt.Errorf("failed to upload file to S3: %w", err)
	}

	return nil
}

// DownloadFile downloads a file from S3-compatible storage
func (s *s3Storage) DownloadFile(ctx context.Context, bucket, key string) ([]byte, error) {
	if bucket == "" {
		bucket = s.bucket
	}

	result, err := s.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		// Check if it's a NoSuchKey error
		if _, ok := err.(*types.NoSuchKey); ok {
			return nil, fmt.Errorf("file not found: %s/%s", bucket, key)
		}
		return nil, fmt.Errorf("failed to download file from S3: %w", err)
	}
	defer result.Body.Close()

	data, err := io.ReadAll(result.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read downloaded file: %w", err)
	}

	return data, nil
}

// DeleteFile deletes a file from S3-compatible storage
func (s *s3Storage) DeleteFile(ctx context.Context, bucket, key string) error {
	if bucket == "" {
		bucket = s.bucket
	}

	_, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return fmt.Errorf("failed to delete file from S3: %w", err)
	}

	return nil
}

// FileExists checks if a file exists in S3-compatible storage
func (s *s3Storage) FileExists(ctx context.Context, bucket, key string) (bool, error) {
	if bucket == "" {
		bucket = s.bucket
	}

	_, err := s.client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		// Check if it's a NoSuchKey error
		if _, ok := err.(*types.NoSuchKey); ok {
			return false, nil
		}
		return false, fmt.Errorf("failed to check if file exists: %w", err)
	}

	return true, nil
}