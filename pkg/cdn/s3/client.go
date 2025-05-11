package s3

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

var (
	ErrInvalidFile     = errors.New("invalid file")
	ErrFileTooLarge    = errors.New("file size exceeds the maximum allowed size")
	ErrInvalidFileType = errors.New("file type not supported")
)

// MaxFileSize is the maximum allowed file size (5MB)
const MaxFileSize = 5 * 1024 * 1024

// AllowedFileTypes is a list of allowed image file extensions
var AllowedFileTypes = []string{".jpg", ".jpeg", ".png", ".gif", ".webp"}

// Service provides image operations using S3/MinIO
type Service struct {
	client        *s3.Client
	bucketName    string
	expiry        time.Duration
	baseURLPrefix string
}

// NewService creates a new S3/MinIO service with the given configuration
func NewService(cfg *Config) (*Service, error) {
	// Create custom resolver to handle MinIO
	customResolver := aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...interface{}) (aws.Endpoint, error) {
		return aws.Endpoint{
			URL:               cfg.Endpoint,
			HostnameImmutable: true,
			SigningRegion:     cfg.Region,
		}, nil
	})

	// Create AWS config
	awsCfg, err := config.LoadDefaultConfig(context.Background(),
		config.WithRegion(cfg.Region),
		config.WithEndpointResolverWithOptions(customResolver),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			cfg.AccessKeyID,
			cfg.SecretAccessKey,
			cfg.SessionToken,
		)),
	)
	if err != nil {
		return nil, fmt.Errorf("unable to load SDK config: %w", err)
	}

	// Create S3 client
	client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		o.UsePathStyle = true // MinIO requires path-style URLs
	})

	// Build base URL prefix for direct URLs
	scheme := "http"
	if cfg.UseSSL {
		scheme = "https"
	}
	baseURLPrefix := fmt.Sprintf("%s://%s/%s/", scheme, strings.TrimPrefix(cfg.Endpoint, scheme+"://"), cfg.BucketName)

	return &Service{
		client:        client,
		bucketName:    cfg.BucketName,
		expiry:        cfg.Expiry,
		baseURLPrefix: baseURLPrefix,
	}, nil
}

// UploadResponse represents the result of an image upload
type UploadResponse struct {
	Key         string
	URL         string
	ContentType string
	Size        int64
}

// UploadProfilePhoto uploads a profile photo for a user
func (s *Service) UploadProfilePhoto(ctx context.Context, userID string, file *multipart.FileHeader) (*UploadResponse, error) {
	key := fmt.Sprintf("user-profiles/%s/profile%s", userID, filepath.Ext(file.Filename))
	return s.uploadPhoto(ctx, userID, file, key)
}

// UploadUserPhoto uploads a regular photo for a user with a slot number
func (s *Service) UploadUserPhoto(ctx context.Context, userID string, file *multipart.FileHeader, slot int) (*UploadResponse, error) {
	key := fmt.Sprintf("user-photos/%s/photo_%d%s", userID, slot, filepath.Ext(file.Filename))
	return s.uploadPhoto(ctx, userID, file, key)
}

// uploadPhoto handles the actual upload process
func (s *Service) uploadPhoto(ctx context.Context, userID string, file *multipart.FileHeader, key string) (*UploadResponse, error) {
	// Validate file size
	if file.Size > MaxFileSize {
		return nil, ErrFileTooLarge
	}

	// Validate file type
	ext := strings.ToLower(filepath.Ext(file.Filename))
	if !isValidFileType(ext) {
		return nil, ErrInvalidFileType
	}

	// Open the uploaded file
	src, err := file.Open()
	if err != nil {
		return nil, fmt.Errorf("opening uploaded file: %w", err)
	}
	defer src.Close()

	// Read file data
	fileData, err := io.ReadAll(src)
	if err != nil {
		return nil, fmt.Errorf("reading file data: %w", err)
	}

	// Determine content type
	contentType := file.Header.Get("Content-Type")
	if contentType == "" {
		contentType = http.DetectContentType(fileData)
	}

	// Upload to S3/MinIO
	_, err = s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(s.bucketName),
		Key:         aws.String(key),
		Body:        bytes.NewReader(fileData),
		ContentType: aws.String(contentType),
	})
	if err != nil {
		return nil, fmt.Errorf("uploading to S3/MinIO: %w", err)
	}

	// Build direct URL
	url := s.baseURLPrefix + key

	return &UploadResponse{
		Key:         key,
		URL:         url,
		ContentType: contentType,
		Size:        file.Size,
	}, nil
}

// GetPresignedURL generates a presigned URL for an object
func (s *Service) GetPresignedURL(ctx context.Context, key string) (string, error) {
	presignClient := s3.NewPresignClient(s.client)

	presignedReq, err := presignClient.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucketName),
		Key:    aws.String(key),
	}, func(opts *s3.PresignOptions) {
		opts.Expires = s.expiry
	})
	if err != nil {
		return "", fmt.Errorf("generating presigned URL: %w", err)
	}

	return presignedReq.URL, nil
}

// DeletePhoto deletes a photo by key
func (s *Service) DeletePhoto(ctx context.Context, key string) error {
	_, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(s.bucketName),
		Key:    aws.String(key),
	})
	if err != nil {
		return fmt.Errorf("deleting from S3/MinIO: %w", err)
	}

	return nil
}

// CheckIfBucketExists checks if the configured bucket exists
func (s *Service) CheckIfBucketExists(ctx context.Context) (bool, error) {
	_, err := s.client.HeadBucket(ctx, &s3.HeadBucketInput{
		Bucket: aws.String(s.bucketName),
	})
	if err != nil {
		var nsBucketNotFound *types.NoSuchBucket
		if errors.As(err, &nsBucketNotFound) {
			return false, nil
		}
		return false, fmt.Errorf("checking if bucket exists: %w", err)
	}
	return true, nil
}

// CreateBucket creates the configured bucket if it doesn't exist
func (s *Service) CreateBucket(ctx context.Context) error {
	exists, err := s.CheckIfBucketExists(ctx)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}

	_, err = s.client.CreateBucket(ctx, &s3.CreateBucketInput{
		Bucket: aws.String(s.bucketName),
	})
	if err != nil {
		return fmt.Errorf("creating bucket: %w", err)
	}

	// Make the bucket public
	_, err = s.client.PutBucketPolicy(ctx, &s3.PutBucketPolicyInput{
		Bucket: aws.String(s.bucketName),
		Policy: aws.String(`{
			"Version": "2012-10-17",
			"Statement": [
				{
					"Effect": "Allow",
					"Principal": "*",
					"Action": "s3:GetObject",
					"Resource": "arn:aws:s3:::` + s.bucketName + `/*"
				}
			]
		}`),
	})
	if err != nil {
		return fmt.Errorf("setting bucket policy: %w", err)
	}

	return nil
}

// Helper function to check if file type is allowed
func isValidFileType(ext string) bool {
	for _, allowedType := range AllowedFileTypes {
		if allowedType == ext {
			return true
		}
	}
	return false
}
