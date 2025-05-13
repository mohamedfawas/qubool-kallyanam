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
var AllowedFileTypes = []string{".jpg", ".jpeg", ".png"}

// isValidFileType checks extension against AllowedFileTypes
func isValidFileType(ext string) bool {
	for _, allowedType := range AllowedFileTypes {
		if allowedType == ext {
			return true
		}
	}
	return false
}

// Service provides image operations using S3/MinIO
// ============================================================================
// SERVICE STRUCT AND INITIALIZATION
// ============================================================================

// Service wraps the AWS S3 client and bucket settings for image operations.
// client        - AWS S3 client used to make API calls.
// bucketName    - The S3 bucket where files are stored.
// expiry        - How long presigned URLs are valid.
// baseURLPrefix - Base URL for direct public links (if bucket is public).
type Service struct {
	client        *s3.Client
	bucketName    string
	expiry        time.Duration
	baseURLPrefix string
}

// NewService initializes the Service using the provided Config.
// It handles both AWS S3 and MinIO (an S3-compatible server).
func NewService(cfg *Config) (*Service, error) {
	// Custom resolver ensures requests go to cfg.Endpoint (e.g. MinIO)
	customResolver := aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...interface{}) (aws.Endpoint, error) {
		return aws.Endpoint{
			URL:               cfg.Endpoint,
			HostnameImmutable: true,
			SigningRegion:     cfg.Region,
		}, nil
	})

	// Load AWS SDK config with region, endpoint override, and credentials
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

	// Create the S3 client; path style is needed for MinIO
	client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		o.UsePathStyle = true // MinIO requires path-style URLs
	})

	// Determine URL scheme (http vs https)
	scheme := "http"
	if cfg.UseSSL {
		scheme = "https"
	}
	// baseURLPrefix is like "https://play.min.io/mybucket/"
	baseURLPrefix := fmt.Sprintf("%s://%s/%s/", scheme, strings.TrimPrefix(cfg.Endpoint, scheme+"://"), cfg.BucketName)

	return &Service{
		client:        client,
		bucketName:    cfg.BucketName,
		expiry:        cfg.Expiry,
		baseURLPrefix: baseURLPrefix,
	}, nil
}

// ============================================================================
// UPLOAD RESPONSE MODEL
// ============================================================================

// UploadResponse is returned after a successful upload.
// Fields:
//
//	Key         - object key (path) in the bucket, e.g. "user-profiles/123/profile.png"
//	URL         - public URL to access the file (if bucket is public)
//	ContentType - MIME type, e.g. "image/png"
//	Size        - size of the file in bytes
type UploadResponse struct {
	Key         string
	URL         string
	ContentType string
	Size        int64
}

// UploadProfilePhoto stores a user's profile image in "user-profiles/{userID}/profile{.ext}".
func (s *Service) UploadProfilePhoto(ctx context.Context, userID string, file *multipart.FileHeader) (*UploadResponse, error) {
	key := fmt.Sprintf("user-profiles/%s/profile%s", userID, filepath.Ext(file.Filename))
	return s.uploadPhoto(ctx, userID, file, key)
}

// UploadUserPhoto uploads a regular photo for a user with a slot number
// func (s *Service) UploadUserPhoto(ctx context.Context, userID string, file *multipart.FileHeader, slot int) (*UploadResponse, error) {
// 	key := fmt.Sprintf("user-photos/%s/photo_%d%s", userID, slot, filepath.Ext(file.Filename))
// 	return s.uploadPhoto(ctx, userID, file, key)
// }

// uploadPhoto performs validation and uploads the file to S3/MinIO.
func (s *Service) uploadPhoto(ctx context.Context, userID string, file *multipart.FileHeader, key string) (*UploadResponse, error) {
	// 1) Validate file size
	if file.Size > MaxFileSize {
		return nil, ErrFileTooLarge
	}

	// 2) Validate file type (extension)
	ext := strings.ToLower(filepath.Ext(file.Filename))
	if !isValidFileType(ext) {
		return nil, ErrInvalidFileType
	}

	// 3) Open the file stream (reader)
	src, err := file.Open()
	if err != nil {
		return nil, fmt.Errorf("opening uploaded file: %w", err)
	}
	// Ensure reader is closed when done
	defer src.Close()

	// 4) Read all data into memory (byte slice)
	fileData, err := io.ReadAll(src)
	if err != nil {
		return nil, fmt.Errorf("reading file data: %w", err)
	}

	// 5) Determine Content-Type header
	contentType := file.Header.Get("Content-Type")
	if contentType == "" {
		// Fallback: auto-detect if header missing
		contentType = http.DetectContentType(fileData)
	}

	// 6) Upload to S3 using PutObject
	_, err = s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(s.bucketName),
		Key:         aws.String(key),
		Body:        bytes.NewReader(fileData),
		ContentType: aws.String(contentType),
	})
	if err != nil {
		return nil, fmt.Errorf("uploading to S3/MinIO: %w", err)
	}

	// 7) Build a direct URL for public access
	url := s.baseURLPrefix + key

	return &UploadResponse{
		Key:         key,
		URL:         url,
		ContentType: contentType,
		Size:        file.Size,
	}, nil
}

// ==============================
// What is a Presigned URL?
// ==============================
// A presigned URL is a secure, time-limited URL that provides temporary access to a private file stored in an Amazon S3 bucket.
// It is "signed" using AWS credentials and permissions, which means the URL can only be used for a short period (e.g., 15 minutes),
// and only to perform a specific operation (like GET or PUT) on a specific file (object).

// ==============================
// Why do we use Presigned URLs?
// ==============================
// - S3 buckets are typically private to protect files from unauthorized access.
// - Sometimes, we want to allow specific users to access a file (e.g., to download a PDF invoice or an image)
//   without making the whole bucket public or giving users full S3 access.
// - A presigned URL allows us to safely give **temporary access** to that private file — only for the needed operation (GET or PUT),
//   and only for a short time (defined by an expiration).

// ==============================
// Example Use Cases:
// ==============================
//  1. A user uploads their profile picture via a mobile app.
//     ➝ Backend generates a presigned URL for uploading (PUT operation).
//     ➝ The mobile app uploads the image directly to S3 using that URL.
//
// ==============================
// Summary:
// ==============================
// Presigned URLs help you share access to private S3 files safely without exposing your AWS credentials,
// and without making your S3 bucket public.
// They're a common pattern for file sharing or uploads in secure backend systems.
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

// DeletePhoto removes an object from the bucket by key.
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

// CheckIfBucketExists returns true if the configured bucket already exists.
func (s *Service) CheckIfBucketExists(ctx context.Context) (bool, error) {
	_, err := s.client.HeadBucket(ctx, &s3.HeadBucketInput{
		Bucket: aws.String(s.bucketName),
	})
	if err != nil {
		// If bucket not found, return false without error
		var nsBucketNotFound *types.NoSuchBucket
		if errors.As(err, &nsBucketNotFound) {
			return false, nil
		}
		return false, fmt.Errorf("checking if bucket exists: %w", err)
	}
	return true, nil
}

// CreateBucket creates the bucket if it does not exist, and makes it public.
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

	// Apply policy to allow public GET of objects
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
