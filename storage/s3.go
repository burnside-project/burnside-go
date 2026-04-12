package storage

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"

	"github.com/dataalgebra-engineering/burnside-project-burnside-go/manifest"
)

// S3 implements ReaderWriter using AWS S3 (or S3-compatible storage like MinIO).
type S3 struct {
	client *s3.Client
	bucket string
	prefix string
}

// NewS3 creates an S3 storage backend.
// prefix should not start with "/" but should end with "/" if non-empty.
func NewS3(client *s3.Client, bucket, prefix string) *S3 {
	prefix = strings.TrimPrefix(prefix, "/")
	if prefix != "" && !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}
	return &S3{client: client, bucket: bucket, prefix: prefix}
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
	return s.writeBytes(ctx, "manifest.json", buf.Bytes(), "application/json")
}

func (s *S3) WriteFile(ctx context.Context, path string, r io.Reader) error {
	data, err := io.ReadAll(r)
	if err != nil {
		return fmt.Errorf("read content for %q: %w", path, err)
	}
	return s.writeBytes(ctx, path, data, "application/octet-stream")
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

func (s *S3) writeBytes(ctx context.Context, path string, data []byte, contentType string) error {
	_, err := s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(s.bucket),
		Key:         aws.String(s.key(path)),
		Body:        bytes.NewReader(data),
		ContentType: aws.String(contentType),
	})
	if err != nil {
		return fmt.Errorf("put s3://%s/%s: %w", s.bucket, s.key(path), err)
	}
	return nil
}

// Ensure S3 implements ReaderWriter.
var _ ReaderWriter = (*S3)(nil)
