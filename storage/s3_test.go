package storage

import (
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

func TestNewS3PrefixNormalisation(t *testing.T) {
	client := s3.New(s3.Options{Region: "us-east-1"})

	tests := []struct {
		in, want string
	}{
		{"", ""},
		{"/cdc/prod", "cdc/prod/"},
		{"cdc/prod", "cdc/prod/"},
		{"cdc/prod/", "cdc/prod/"},
		{"/cdc/prod/", "cdc/prod/"},
	}

	for _, tc := range tests {
		s := NewS3(client, "bucket", tc.in)
		if s.prefix != tc.want {
			t.Errorf("NewS3(prefix=%q): got prefix=%q, want %q", tc.in, s.prefix, tc.want)
		}
		if got := s.key("foo.parquet"); got != tc.want+"foo.parquet" {
			t.Errorf("NewS3(prefix=%q).key(\"foo.parquet\"): got %q, want %q", tc.in, got, tc.want+"foo.parquet")
		}
	}
}

func TestNewS3Defaults(t *testing.T) {
	client := s3.New(s3.Options{Region: "us-east-1"})
	s := NewS3(client, "bucket", "")

	if s.partSize != defaultUploadPartSize {
		t.Errorf("default partSize: got %d, want %d", s.partSize, defaultUploadPartSize)
	}
	if s.concurrency != defaultUploadConcurrency {
		t.Errorf("default concurrency: got %d, want %d", s.concurrency, defaultUploadConcurrency)
	}
	if s.transfer == nil {
		t.Error("transfer manager not initialised")
	}
}

func TestNewS3WithOptions(t *testing.T) {
	client := s3.New(s3.Options{Region: "us-east-1"})
	s := NewS3(client, "bucket", "",
		WithPartSize(16*1024*1024),
		WithUploadConcurrency(8),
	)

	if s.partSize != 16*1024*1024 {
		t.Errorf("partSize: got %d, want %d", s.partSize, 16*1024*1024)
	}
	if s.concurrency != 8 {
		t.Errorf("concurrency: got %d, want %d", s.concurrency, 8)
	}
}

// Compile-time check that NewS3 returns a ReaderWriter.
func TestS3ImplementsReaderWriter(t *testing.T) {
	client := s3.New(s3.Options{Region: "us-east-1"})
	var _ ReaderWriter = NewS3(client, "bucket", "")
	_ = aws.String // keep import used in future assertions
}
