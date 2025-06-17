package storage

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
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
	DeleteProfilePhoto(ctx context.Context, userID uuid.UUID) error
	EnsureBucketExists(ctx context.Context) error
	UploadUserPhoto(ctx context.Context, userID uuid.UUID, header *multipart.FileHeader, file io.Reader, displayOrder int) (string, string, error) // returns (photoURL, photoKey, error)
	DeleteUserPhoto(ctx context.Context, photoKey string) error
	UploadUserVideo(ctx context.Context, userID uuid.UUID, header *multipart.FileHeader, file io.Reader) (string, string, error) // returns (videoURL, videoKey, error)
	DeleteUserVideo(ctx context.Context, videoKey string) error
}

// S3PhotoStorage implements PhotoStorage using AWS S3 or MinIO.
// It holds an S3 client, bucket name, base URL, and a logger for diagnostics.
// In production, it points to AWS S3; in development, you can point Endpoint to a MinIO instance.
type S3PhotoStorage struct {
	s3Client   *s3.Client // S3 client to perform API calls
	bucketName string     // Name of the S3 bucket to store photos
	baseURL    string     // Base URL to construct public URLs for objects
	logger     logging.Logger
}

// NewS3PhotoStorage creates a new S3PhotoStorage
func NewS3PhotoStorage(s3Config *s3pkg.Config, logger logging.Logger) (*S3PhotoStorage, error) {
	// Define how to resolve the endpoint (where to send the request, e.g., MinIo or AWS S3)
	customResolver := aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...interface{}) (aws.Endpoint, error) {
		return aws.Endpoint{
			URL:               s3Config.Endpoint, // Custom URL (like https://<bucket-name>.s3.<region>.amazonaws.com) // e.g., "http://localhost:9000"
			HostnameImmutable: true,              // prevent SDK from modifying the hostname
			SigningRegion:     s3Config.Region,   // region for request signing
		}, nil
	})

	// Load AWS config with credentials and custom endpoint
	awsCfg, err := config.LoadDefaultConfig(context.Background(),
		config.WithRegion(s3Config.Region),
		config.WithEndpointResolverWithOptions(customResolver),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			s3Config.AccessKeyID,
			s3Config.SecretAccessKey,
			"", // Session token (usually blank if not temporary credentials)
			/*
							A session token is a temporary security credential used only when you're using temporary access to AWS, like:

				When you use IAM roles with AWS STS (Security Token Service).

				When you use federated login (like Google or GitHub SSO to log in to AWS).

				When you assume a role using tools like the AWS CLI.

				These tokens expire after a certain time, unlike access keys which are usually long-lived.
			*/
		)),
	)
	if err != nil {
		// Failed to load AWS config (check your keys or endpoint)
		return nil, fmt.Errorf("unable to load SDK config: %w", err)
	}

	// Create the S3 client with PathStyle enabled (necessary for MinIO)
	client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		o.UsePathStyle = true
	})

	// Determine scheme (http or https) for generating URLs later.
	scheme := "http"
	if s3Config.UseSSL {
		scheme = "https"
	}
	// Use public URL from environment if set, otherwise fall back to endpoint
	baseURL := os.Getenv("S3_PUBLIC_URL")
	if baseURL == "" {
		baseURL = fmt.Sprintf("%s://%s", scheme, strings.TrimPrefix(s3Config.Endpoint, scheme+"://"))
	}
	baseURL = fmt.Sprintf("%s/%s/", baseURL, s3Config.BucketName)

	return &S3PhotoStorage{
		s3Client:   client,
		bucketName: s3Config.BucketName,
		baseURL:    baseURL,
		logger:     logger,
	}, nil
}

// UploadProfilePhoto uploads a user's profile picture to the bucket.
// Steps:
// 1. Read all bytes from the file reader.
// 2. Validate size (< 5MB) and extension (.jpg, .png, etc.).
// 3. Detect MIME content type (e.g., "image/jpeg").
// 4. Construct the object key: "user-profiles/{userID}/profile.ext".
// 5. Call PutObject to upload.
// 6. Return the public URL: baseURL + key.
func (s *S3PhotoStorage) UploadProfilePhoto(ctx context.Context, userID uuid.UUID, header *multipart.FileHeader, file io.Reader) (string, error) {
	s.logger.Info("Uploading profile photo", "userID", userID.String())

	// 1. Read full file into memory (not streaming).
	fileData, err := io.ReadAll(file)
	if err != nil {
		s.logger.Error("Failed to read file data", "error", err, "userID", userID.String())
		return "", fmt.Errorf("failed to read file data: %w", err)
	}

	// 2. Size check: limit to 5 * 1024 * 1024 bytes = 5MB.
	if len(fileData) > 5*1024*1024 {
		s.logger.Error("File too large", "size", len(fileData), "userID", userID.String())
		return "", fmt.Errorf("file size exceeds the maximum allowed size of 5MB")
	}

	// 3. Extension check: allow common image types.
	ext := strings.ToLower(filepath.Ext(header.Filename))
	validExts := []string{".jpg", ".jpeg", ".png"}
	isValidExt := false
	for _, validExt := range validExts {
		if ext == validExt {
			isValidExt = true
			break
		}
	}
	if !isValidExt {
		s.logger.Error("Invalid file type", "extension", ext, "userID", userID.String())
		return "", fmt.Errorf("unsupported file type. Allowed types: jpg, jpeg, png")
	}

	// 4. Detect the MIME type from the data (safer than trusting extension)
	contentType := http.DetectContentType(fileData) // e.g., "image/jpeg"

	// 5. Build S3 object key. Example: "user-profiles/123e4567-e89b-12d3-a456-426614174000/profile.png"
	key := fmt.Sprintf("user-profiles/%s/profile%s", userID.String(), ext)

	// 6. Upload the object to S3.
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

	// 7. Construct public URL for access: baseURL + key
	photoURL := s.baseURL + key
	s.logger.Info("Successfully uploaded profile photo", "userID", userID.String(), "url", photoURL)
	return photoURL, nil
}

// DeleteProfilePhoto removes all objects under user-profiles/{userID}/
// Steps:
// 1. List objects with that prefix.
// 2. If none, do nothing.
// 3. Delete each found object.
func (s *S3PhotoStorage) DeleteProfilePhoto(ctx context.Context, userID uuid.UUID) error {
	s.logger.Info("Deleting profile photo", "userID", userID.String())

	// Prefix to search within the bucket: folder per user.
	prefix := fmt.Sprintf("user-profiles/%s/", userID.String())
	listOutput, err := s.s3Client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
		Bucket: aws.String(s.bucketName),
		Prefix: aws.String(prefix),
	})

	if err != nil {
		s.logger.Error("Failed to list objects in S3", "error", err, "userID", userID.String())
		return fmt.Errorf("failed to list objects in S3: %w", err)
	}

	// If no items, nothing to delete.
	if len(listOutput.Contents) == 0 {
		s.logger.Info("No profile photo found to delete", "userID", userID.String())
		return nil
	}

	// Iterate and delete each object found.
	for _, obj := range listOutput.Contents {
		_, err = s.s3Client.DeleteObject(ctx, &s3.DeleteObjectInput{
			Bucket: aws.String(s.bucketName),
			Key:    obj.Key,
		})

		if err != nil {
			s.logger.Error("Failed to delete object from S3", "error", err, "userID", userID.String(), "key", *obj.Key)
			return fmt.Errorf("failed to delete object from S3: %w", err)
		}

		s.logger.Info("Successfully deleted object", "userID", userID.String(), "key", *obj.Key)
	}

	s.logger.Info("Successfully deleted profile photo", "userID", userID.String())
	return nil
}

// EnsureBucketExists checks for the bucket and creates it if missing.
// It also applies a public-read policy so uploaded images are publicly accessible.
func (s *S3PhotoStorage) EnsureBucketExists(ctx context.Context) error {
	// Try to get bucket metadata. If it fails, assume missing.
	_, err := s.s3Client.HeadBucket(ctx, &s3.HeadBucketInput{
		Bucket: aws.String(s.bucketName),
	})
	if err != nil {
		// Bucket doesn't exist; create it.
		s.logger.Info("Bucket doesn't exist, creating new bucket", "bucket", s.bucketName)
		_, err = s.s3Client.CreateBucket(ctx, &s3.CreateBucketInput{
			Bucket: aws.String(s.bucketName),
		})
		if err != nil {
			s.logger.Error("Failed to create bucket", "error", err)
			return fmt.Errorf("failed to create bucket: %w", err)
		}

		// Define public-read policy so all objects under the bucket can be fetched by anyone.
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

		// Apply bucket policy
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

// UploadUserPhoto uploads an additional photo for a user
func (s *S3PhotoStorage) UploadUserPhoto(ctx context.Context, userID uuid.UUID, header *multipart.FileHeader, file io.Reader, displayOrder int) (string, string, error) {
	s.logger.Info("Uploading user photo", "userID", userID.String(), "displayOrder", displayOrder)

	// Read file data
	fileData, err := io.ReadAll(file)
	if err != nil {
		s.logger.Error("Failed to read file data", "error", err, "userID", userID.String())
		return "", "", fmt.Errorf("failed to read file data: %w", err)
	}

	// Size check: 5MB limit
	if len(fileData) > 5*1024*1024 {
		s.logger.Error("File too large", "size", len(fileData), "userID", userID.String())
		return "", "", fmt.Errorf("file size exceeds the maximum allowed size of 5MB")
	}

	// Extension check
	ext := strings.ToLower(filepath.Ext(header.Filename))
	validExts := []string{".jpg", ".jpeg", ".png"}
	isValidExt := false
	for _, validExt := range validExts {
		if ext == validExt {
			isValidExt = true
			break
		}
	}
	if !isValidExt {
		s.logger.Error("Invalid file type", "extension", ext, "userID", userID.String())
		return "", "", fmt.Errorf("unsupported file type. Allowed types: jpg, jpeg, png")
	}

	// Detect content type
	contentType := http.DetectContentType(fileData)

	// Build S3 object key: "user-photos/{userID}/photo_{displayOrder}.ext"
	key := fmt.Sprintf("user-photos/%s/photo_%d%s", userID.String(), displayOrder, ext)

	// Upload to S3
	_, err = s.s3Client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(s.bucketName),
		Key:         aws.String(key),
		Body:        bytes.NewReader(fileData),
		ContentType: aws.String(contentType),
	})
	if err != nil {
		s.logger.Error("Failed to upload to S3", "error", err, "userID", userID.String())
		return "", "", fmt.Errorf("failed to upload to S3: %w", err)
	}

	// Construct public URL
	photoURL := s.baseURL + key
	s.logger.Info("Successfully uploaded user photo", "userID", userID.String(), "displayOrder", displayOrder, "url", photoURL)
	return photoURL, key, nil
}

// DeleteUserPhoto deletes a specific user photo by its key
func (s *S3PhotoStorage) DeleteUserPhoto(ctx context.Context, photoKey string) error {
	s.logger.Info("Deleting user photo", "key", photoKey)

	_, err := s.s3Client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(s.bucketName),
		Key:    aws.String(photoKey),
	})

	if err != nil {
		s.logger.Error("Failed to delete user photo from S3", "error", err, "key", photoKey)
		return fmt.Errorf("failed to delete user photo from S3: %w", err)
	}

	s.logger.Info("Successfully deleted user photo", "key", photoKey)
	return nil
}

// UploadUserVideo uploads an introduction video for a user
func (s *S3PhotoStorage) UploadUserVideo(ctx context.Context, userID uuid.UUID, header *multipart.FileHeader, file io.Reader) (string, string, error) {
	s.logger.Info("Uploading user video", "userID", userID.String())

	// Read file data
	fileData, err := io.ReadAll(file)
	if err != nil {
		s.logger.Error("Failed to read video file data", "error", err, "userID", userID.String())
		return "", "", fmt.Errorf("failed to read video file data: %w", err)
	}

	// Size check: 50MB limit for 1-minute video
	if len(fileData) > 50*1024*1024 {
		s.logger.Error("Video file too large", "size", len(fileData), "userID", userID.String())
		return "", "", fmt.Errorf("video file size exceeds the maximum allowed size of 50MB")
	}

	// Extension check
	ext := strings.ToLower(filepath.Ext(header.Filename))
	validExts := []string{".mp4", ".mov", ".avi", ".mkv"}
	isValidExt := false
	for _, validExt := range validExts {
		if ext == validExt {
			isValidExt = true
			break
		}
	}
	if !isValidExt {
		s.logger.Error("Invalid video file type", "extension", ext, "userID", userID.String())
		return "", "", fmt.Errorf("unsupported video file type. Allowed types: mp4, mov, avi, mkv")
	}

	// Detect content type
	contentType := header.Header.Get("Content-Type")
	if contentType == "" {
		// Set default content type based on extension
		switch ext {
		case ".mp4":
			contentType = "video/mp4"
		case ".mov":
			contentType = "video/quicktime"
		case ".avi":
			contentType = "video/x-msvideo"
		case ".mkv":
			contentType = "video/x-matroska"
		default:
			contentType = "video/mp4"
		}
	}

	// Build S3 object key: "user-videos/{userID}/intro_video.ext"
	key := fmt.Sprintf("user-videos/%s/intro_video%s", userID.String(), ext)

	// Upload to S3
	_, err = s.s3Client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(s.bucketName),
		Key:         aws.String(key),
		Body:        bytes.NewReader(fileData),
		ContentType: aws.String(contentType),
	})
	if err != nil {
		s.logger.Error("Failed to upload video to S3", "error", err, "userID", userID.String())
		return "", "", fmt.Errorf("failed to upload video to S3: %w", err)
	}

	// Construct public URL
	videoURL := s.baseURL + key
	s.logger.Info("Successfully uploaded user video", "userID", userID.String(), "url", videoURL)
	return videoURL, key, nil
}

// DeleteUserVideo deletes a specific user video by its key
func (s *S3PhotoStorage) DeleteUserVideo(ctx context.Context, videoKey string) error {
	s.logger.Info("Deleting user video", "key", videoKey)

	_, err := s.s3Client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(s.bucketName),
		Key:    aws.String(videoKey),
	})

	if err != nil {
		s.logger.Error("Failed to delete user video from S3", "error", err, "key", videoKey)
		return fmt.Errorf("failed to delete user video from S3: %w", err)
	}

	s.logger.Info("Successfully deleted user video", "key", videoKey)
	return nil
}
