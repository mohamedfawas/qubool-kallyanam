package storage

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/google/uuid"
	s3pkg "github.com/mohamedfawas/qubool-kallyanam/pkg/cdn/s3"
	"github.com/mohamedfawas/qubool-kallyanam/pkg/logging"
)

// PhotoStorage defines the interface for photo storage operations
type PhotoStorage interface {
	UploadProfilePhoto(ctx context.Context, userID uuid.UUID, header *multipart.FileHeader, file io.Reader) (string, error)
	EnsureBucketExists(ctx context.Context) error
}

// S3PhotoStorage implements the PhotoStorage interface
type S3PhotoStorage struct {
	s3Client   *s3.Client
	bucketName string
	baseURL    string
	logger     logging.Logger
}

// NewS3PhotoStorage creates a new S3PhotoStorage
func NewS3PhotoStorage(s3Config *s3pkg.Config, logger logging.Logger) (*S3PhotoStorage, error) {
	// Create custom resolver for endpoint
	customResolver := aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...interface{}) (aws.Endpoint, error) {
		return aws.Endpoint{
			URL:               s3Config.Endpoint,
			HostnameImmutable: true,
			SigningRegion:     s3Config.Region,
		}, nil
	})

	// Load AWS config
	awsCfg, err := config.LoadDefaultConfig(context.Background(),
		config.WithRegion(s3Config.Region),
		config.WithEndpointResolverWithOptions(customResolver),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			s3Config.AccessKeyID,
			s3Config.SecretAccessKey,
			"", // Session token
		)),
	)
	if err != nil {
		return nil, fmt.Errorf("unable to load SDK config: %w", err)
	}

	// Create S3 client
	client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		o.UsePathStyle = true
	})

	// Generate base URL
	scheme := "http"
	if s3Config.UseSSL {
		scheme = "https"
	}
	baseURL := fmt.Sprintf("%s://%s/%s/",
		scheme,
		strings.TrimPrefix(s3Config.Endpoint, scheme+"://"),
		s3Config.BucketName)

	return &S3PhotoStorage{
		s3Client:   client,
		bucketName: s3Config.BucketName,
		baseURL:    baseURL,
		logger:     logger,
	}, nil
}

// UploadProfilePhoto uploads a profile photo to S3
func (s *S3PhotoStorage) UploadProfilePhoto(ctx context.Context, userID uuid.UUID, header *multipart.FileHeader, file io.Reader) (string, error) {
	s.logger.Info("Uploading profile photo", "userID", userID.String())

	// Read file data
	fileData, err := io.ReadAll(file)
	if err != nil {
		s.logger.Error("Failed to read file data", "error", err, "userID", userID.String())
		return "", fmt.Errorf("failed to read file data: %w", err)
	}

	// Validate file size
	if len(fileData) > 5*1024*1024 {
		s.logger.Error("File too large", "size", len(fileData), "userID", userID.String())
		return "", fmt.Errorf("file size exceeds the maximum allowed size of 5MB")
	}

	// Validate file extension
	ext := strings.ToLower(filepath.Ext(header.Filename))
	validExts := []string{".jpg", ".jpeg", ".png", ".gif", ".webp"}
	isValidExt := false
	for _, validExt := range validExts {
		if ext == validExt {
			isValidExt = true
			break
		}
	}
	if !isValidExt {
		s.logger.Error("Invalid file type", "extension", ext, "userID", userID.String())
		return "", fmt.Errorf("unsupported file type. Allowed types: jpg, jpeg, png, gif, webp")
	}

	// Detect content type
	contentType := http.DetectContentType(fileData)

	// Create a key for the file
	key := fmt.Sprintf("user-profiles/%s/profile%s", userID.String(), ext)

	// Upload to S3
	_, err = s.s3Client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(s.bucketName),
		Key:         aws.String(key),
		Body:        bytes.NewReader(fileData),
		ContentType: aws.String(contentType),
	})
	if err != nil {
		s.logger.Error("Failed to upload to S3", "error", err, "userID", userID.String())
		return "", fmt.Errorf("failed to upload to S3: %w", err)
	}

	// Generate URL
	photoURL := s.baseURL + key
	s.logger.Info("Successfully uploaded profile photo", "userID", userID.String(), "url", photoURL)
	return photoURL, nil
}

// EnsureBucketExists checks if the bucket exists and creates it if it doesn't
func (s *S3PhotoStorage) EnsureBucketExists(ctx context.Context) error {
	// Check if bucket exists
	_, err := s.s3Client.HeadBucket(ctx, &s3.HeadBucketInput{
		Bucket: aws.String(s.bucketName),
	})
	if err != nil {
		// If bucket doesn't exist, create it
		s.logger.Info("Bucket doesn't exist, creating new bucket", "bucket", s.bucketName)
		_, err = s.s3Client.CreateBucket(ctx, &s3.CreateBucketInput{
			Bucket: aws.String(s.bucketName),
		})
		if err != nil {
			s.logger.Error("Failed to create bucket", "error", err)
			return fmt.Errorf("failed to create bucket: %w", err)
		}

		// Set bucket policy to allow public read
		policy := fmt.Sprintf(`{
			"Version": "2012-10-17",
			"Statement": [
				{
					"Effect": "Allow",
					"Principal": "*",
					"Action": "s3:GetObject",
					"Resource": "arn:aws:s3:::%s/*"
				}
			]
		}`, s.bucketName)

		_, err = s.s3Client.PutBucketPolicy(ctx, &s3.PutBucketPolicyInput{
			Bucket: aws.String(s.bucketName),
			Policy: aws.String(policy),
		})
		if err != nil {
			s.logger.Error("Failed to set bucket policy", "error", err)
			return fmt.Errorf("failed to set bucket policy: %w", err)
		}

		s.logger.Info("Successfully created bucket with public read policy", "bucket", s.bucketName)
	} else {
		s.logger.Info("Bucket already exists", "bucket", s.bucketName)
	}

	return nil
}
