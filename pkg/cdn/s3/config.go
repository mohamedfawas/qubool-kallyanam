package s3

import "time"

// ============================================================================
// CONFIGURATION STRUCT AND CONSTRUCTORS
// ============================================================================

// Config holds the configuration needed to connect to AWS S3 or a MinIO server.
// Fields:
//   Endpoint        - URL of the S3/MinIO server, e.g. "play.min.io:9000"
//   Region          - AWS region, e.g. "us-east-1"
//   AccessKeyID     - Your access key ID, like a username for S3 access
//   SecretAccessKey - Your secret key, like a password for S3 access
//   UseSSL          - true if you want HTTPS, false for HTTP
//   BucketName      - Name of the bucket (folder) where you will store files
//   SessionToken    - Optional token for temporary sessions (leave blank for MinIO)
//   Expiry          - How long presigned URLs should remain valid
// Example:
//   cfg := NewConfig("play.min.io:9000", "us-east-1", "ACCESSKEY", "SECRETKEY", "mybucket", true)
//   // By default, Expiry is 24 hours, but you can change with WithExpiry:
//   cfg = cfg.WithExpiry(2 * time.Hour)
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

// WithExpiry allows you to set a custom expiry duration for presigned URLs.
// Returns the same Config pointer so calls can be chained.
// Example:
//   cfg := NewConfig(...).WithExpiry(6 * time.Hour)  // URLs valid for 6 hours
func (c *Config) WithExpiry(expiry time.Duration) *Config {
	c.Expiry = expiry
	return c
}
