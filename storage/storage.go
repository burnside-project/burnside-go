// Package storage defines the interface for reading and writing CDC output
// (Parquet files + manifest) from a storage backend.
package storage

import (
	"context"
	"io"

	"github.com/dataalgebra-engineering/burnside-project-burnside-go/manifest"
)

// Reader provides read access to a CDC output location.
// Implemented by filesystem, S3, and GCS adapters.
type Reader interface {
	// ReadManifest reads and parses manifest.json from the storage root.
	ReadManifest(ctx context.Context) (*manifest.Manifest, error)

	// ListFiles returns all file paths matching the given prefix (relative paths).
	ListFiles(ctx context.Context, prefix string) ([]string, error)

	// ReadFile opens a file for reading. Caller must close the returned ReadCloser.
	ReadFile(ctx context.Context, path string) (io.ReadCloser, error)
}

// Writer provides write access to a CDC output location.
// Implemented by filesystem, S3, and GCS adapters.
type Writer interface {
	// WriteManifest serializes and writes manifest.json to the storage root.
	WriteManifest(ctx context.Context, m *manifest.Manifest) error

	// WriteFile writes content to the given path. Creates parent directories as needed.
	WriteFile(ctx context.Context, path string, r io.Reader) error

	// DeleteFile removes a file at the given path. Returns nil if file doesn't exist.
	DeleteFile(ctx context.Context, path string) error
}

// ReaderWriter combines both read and write access.
type ReaderWriter interface {
	Reader
	Writer
}
