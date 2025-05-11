package s3

import "time"

// Config holds the S3/MinIO configuration
type Config struct {
	Endpoint        string
	Region          string
	AccessKeyID     string
	SecretAccessKey string
	UseSSL          bool
	BucketName      string
	// If using AWS S3, these can be empty for MinIO
	SessionToken string
	Expiry       time.Duration
}

// NewConfig creates a new S3/MinIO configuration
func NewConfig(endpoint, region, accessKeyID, secretAccessKey, bucketName string, useSSL bool) *Config {
	return &Config{
		Endpoint:        endpoint,
		Region:          region,
		AccessKeyID:     accessKeyID,
		SecretAccessKey: secretAccessKey,
		UseSSL:          useSSL,
		BucketName:      bucketName,
		Expiry:          time.Hour * 24, // Default expiry for presigned URLs
	}
}

// WithExpiry sets the expiry duration for presigned URLs
func (c *Config) WithExpiry(expiry time.Duration) *Config {
	c.Expiry = expiry
	return c
}
