package storage

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/s3/transfermanager"
	"github.com/aws/aws-sdk-go-v2/service/s3"

	"github.com/burnside-project/burnside-go/manifest"
)

// Default multipart upload settings. PartSize of 64 MiB and Concurrency of 4
// keep peak memory around ~256 MiB per concurrent upload, a reasonable ceiling
// for CDC hosts streaming 800 MiB+ base files.
const (
	defaultUploadPartSize    = 64 * 1024 * 1024
	defaultUploadConcurrency = 4
)

// S3 implements ReaderWriter using AWS S3 (or S3-compatible storage like MinIO).
type S3 struct {
	client      *s3.Client
	transfer    *transfermanager.Client
	bucket      string
	prefix      string
	partSize    int64
	concurrency int
}

// S3Option configures optional S3 behavior.
type S3Option func(*S3)

// WithPartSize overrides the multipart upload part size in bytes.
// The AWS SDK enforces a 5 MiB minimum.
func WithPartSize(bytes int64) S3Option {
	return func(s *S3) { s.partSize = bytes }
}

// WithUploadConcurrency overrides how many parts are uploaded in parallel.
func WithUploadConcurrency(n int) S3Option {
	return func(s *S3) { s.concurrency = n }
}

// NewS3 creates an S3 storage backend.
// prefix should not start with "/" but should end with "/" if non-empty.
//
// WriteFile streams via the S3 transfer manager so callers can pass
// arbitrarily large readers (e.g. 800 MiB+ Parquet base files) without
// buffering the whole body in memory. WriteManifest uses a single PutObject
// since manifests are small.
//
// Failed multipart uploads are aborted automatically by the transfer manager.
// Operators should still configure a bucket lifecycle rule to clean up
// incomplete multipart uploads older than 1 day as a safety net against
// process crashes.
func NewS3(client *s3.Client, bucket, prefix string, opts ...S3Option) *S3 {
	prefix = strings.TrimPrefix(prefix, "/")
	if prefix != "" && !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}
	s := &S3{
		client:      client,
		bucket:      bucket,
		prefix:      prefix,
		partSize:    defaultUploadPartSize,
		concurrency: defaultUploadConcurrency,
	}
	for _, opt := range opts {
		opt(s)
	}
	s.transfer = transfermanager.New(client, func(o *transfermanager.Options) {
		o.PartSizeBytes = s.partSize
		o.Concurrency = s.concurrency
	})
	return s
}

func (s *S3) key(path string) string {
	return s.prefix + path
}

func (s *S3) ReadManifest(ctx context.Context) (*manifest.Manifest, error) {
	rc, err := s.ReadFile(ctx, "manifest.json")
	if err != nil {
		return nil, fmt.Errorf("read manifest: %w", err)
	}
	defer func() { _ = rc.Close() }()
	return manifest.Read(rc)
}

func (s *S3) ListFiles(ctx context.Context, prefix string) ([]string, error) {
	fullPrefix := s.key(prefix)

	input := &s3.ListObjectsV2Input{
		Bucket: aws.String(s.bucket),
		Prefix: aws.String(fullPrefix),
	}

	var files []string
	paginator := s3.NewListObjectsV2Paginator(s.client, input)
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("list s3://%s/%s: %w", s.bucket, fullPrefix, err)
		}
		for _, obj := range page.Contents {
			// Return paths relative to the storage root (strip our prefix)
			rel := strings.TrimPrefix(aws.ToString(obj.Key), s.prefix)
			files = append(files, rel)
		}
	}

	return files, nil
}

func (s *S3) ReadFile(ctx context.Context, path string) (io.ReadCloser, error) {
	output, err := s.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(s.key(path)),
	})
	if err != nil {
		return nil, fmt.Errorf("get s3://%s/%s: %w", s.bucket, s.key(path), err)
	}
	return output.Body, nil
}

func (s *S3) WriteManifest(ctx context.Context, m *manifest.Manifest) error {
	var buf bytes.Buffer
	if err := m.Write(&buf); err != nil {
		return fmt.Errorf("serialize manifest: %w", err)
	}
	_, err := s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(s.bucket),
		Key:         aws.String(s.key("manifest.json")),
		Body:        bytes.NewReader(buf.Bytes()),
		ContentType: aws.String("application/json"),
	})
	if err != nil {
		return fmt.Errorf("put s3://%s/%s: %w", s.bucket, s.key("manifest.json"), err)
	}
	return nil
}

func (s *S3) WriteFile(ctx context.Context, path string, r io.Reader) error {
	_, err := s.transfer.UploadObject(ctx, &transfermanager.UploadObjectInput{
		Bucket:      aws.String(s.bucket),
		Key:         aws.String(s.key(path)),
		Body:        r,
		ContentType: aws.String("application/octet-stream"),
	})
	if err != nil {
		return fmt.Errorf("upload s3://%s/%s: %w", s.bucket, s.key(path), err)
	}
	return nil
}

func (s *S3) DeleteFile(ctx context.Context, path string) error {
	_, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(s.key(path)),
	})
	if err != nil {
		return fmt.Errorf("delete s3://%s/%s: %w", s.bucket, s.key(path), err)
	}
	return nil
}

// Ensure S3 implements ReaderWriter.
var _ ReaderWriter = (*S3)(nil)
