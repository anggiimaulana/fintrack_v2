package storage

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/smithy-go"
)

// R2Config configures the Cloudflare R2 (S3-compatible) driver.
type R2Config struct {
	// AccountID is the Cloudflare account id used to derive the default
	// endpoint when Endpoint is empty.
	AccountID string
	// Endpoint overrides the derived R2 endpoint (e.g. a local MinIO for tests).
	Endpoint string
	// Region defaults to "auto" (required by R2).
	Region string
	// AccessKeyID / SecretAccessKey are the R2 API token credentials.
	AccessKeyID     string
	SecretAccessKey string
	// Bucket is the target bucket name.
	Bucket string
	// Prefix is an optional key prefix inside the bucket.
	Prefix string
	// PublicBaseURL, when set, is used to build public object URLs. When empty,
	// objects are served through the API instead (uniform with the local driver).
	PublicBaseURL string
}

// Enabled reports whether the minimum R2 configuration is present.
func (c R2Config) Enabled() bool {
	return c.Bucket != "" && c.AccessKeyID != "" && c.SecretAccessKey != "" &&
		(c.Endpoint != "" || c.AccountID != "")
}

// EndpointURL returns the effective S3 endpoint.
func (c R2Config) EndpointURL() string {
	if c.Endpoint != "" {
		return c.Endpoint
	}
	if c.AccountID != "" {
		return fmt.Sprintf("https://%s.r2.cloudflarestorage.com", c.AccountID)
	}
	return ""
}

// R2Store stores objects in a Cloudflare R2 bucket via the S3 API.
type R2Store struct {
	client   *s3.Client
	bucket   string
	prefix   string
	public   string
	endpoint string
}

// NewR2Store builds an R2-backed store from cfg.
func NewR2Store(ctx context.Context, cfg R2Config) (*R2Store, error) {
	if !cfg.Enabled() {
		return nil, fmt.Errorf("%w: r2 bucket/credentials/endpoint incomplete", ErrDisabled)
	}
	region := cfg.Region
	if region == "" {
		region = "auto"
	}
	awsCfg, err := awsconfig.LoadDefaultConfig(ctx,
		awsconfig.WithRegion(region),
		awsconfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			cfg.AccessKeyID, cfg.SecretAccessKey, "")),
	)
	if err != nil {
		return nil, FormatError(DriverR2, err)
	}
	client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		o.BaseEndpoint = aws.String(cfg.EndpointURL())
		o.UsePathStyle = true
	})
	return &R2Store{
		client:   client,
		bucket:   cfg.Bucket,
		prefix:   strings.Trim(cfg.Prefix, "/"),
		public:   strings.TrimRight(cfg.PublicBaseURL, "/"),
		endpoint: cfg.EndpointURL(),
	}, nil
}

// Name returns the driver name.
func (s *R2Store) Name() string { return DriverR2 }

// PublicBaseURL returns the configured public URL prefix ("" when unset).
func (s *R2Store) PublicBaseURL() string { return s.public }

// Bucket returns the configured bucket name.
func (s *R2Store) Bucket() string { return s.bucket }

// Endpoint returns the effective S3 endpoint URL.
func (s *R2Store) Endpoint() string { return s.endpoint }

// Prefix returns the configured key prefix.
func (s *R2Store) Prefix() string { return s.prefix }

func (s *R2Store) objectKey(key string) string {
	key = strings.TrimLeft(key, "/")
	if s.prefix == "" {
		return key
	}
	return s.prefix + "/" + key
}

// Put uploads content under key.
func (s *R2Store) Put(ctx context.Context, key string, content []byte, contentType string) error {
	in := &s3.PutObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(s.objectKey(key)),
		Body:   bytes.NewReader(content),
	}
	if contentType != "" {
		in.ContentType = aws.String(contentType)
	}
	if _, err := s.client.PutObject(ctx, in); err != nil {
		return FormatError(DriverR2, err)
	}
	return nil
}

// Open returns the object for reading.
func (s *R2Store) Open(ctx context.Context, key string) (*Object, error) {
	out, err := s.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(s.objectKey(key)),
	})
	if err != nil {
		if isNotFound(err) {
			return nil, ErrNotFound
		}
		return nil, FormatError(DriverR2, err)
	}
	ct := ContentTypeFor(key)
	if out.ContentType != nil && *out.ContentType != "" {
		ct = *out.ContentType
	}
	var size int64
	if out.ContentLength != nil {
		size = *out.ContentLength
	}
	var mod time.Time
	if out.LastModified != nil {
		mod = *out.LastModified
	}
	return &Object{
		Body:        out.Body,
		ContentType: ct,
		Size:        size,
		ModTime:     mod,
	}, nil
}

// Delete removes an object; a missing object is not an error.
func (s *R2Store) Delete(ctx context.Context, key string) error {
	if _, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(s.objectKey(key)),
	}); err != nil {
		if isNotFound(err) {
			return nil
		}
		return FormatError(DriverR2, err)
	}
	return nil
}

// Healthy verifies the bucket is reachable with the configured credentials.
func (s *R2Store) Healthy(ctx context.Context) error {
	_, err := s.client.HeadBucket(ctx, &s3.HeadBucketInput{Bucket: aws.String(s.bucket)})
	if err != nil {
		return FormatError(DriverR2, err)
	}
	return nil
}

// isNotFound reports whether err is an S3 "no such key"/404 style error.
func isNotFound(err error) bool {
	var nsk *types.NoSuchKey
	if errors.As(err, &nsk) {
		return true
	}
	var nf *types.NotFound
	if errors.As(err, &nf) {
		return true
	}
	var apiErr smithy.APIError
	if errors.As(err, &apiErr) {
		switch apiErr.ErrorCode() {
		case "NoSuchKey", "NotFound", "404":
			return true
		}
	}
	return false
}

// compile-time assertion.
var _ Store = (*R2Store)(nil)
