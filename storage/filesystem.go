package storage

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/dataalgebra-engineering/burnside-project-burnside-go/manifest"
)

// Filesystem implements ReaderWriter using the local filesystem.
type Filesystem struct {
	basePath string
}

// NewFilesystem creates a storage backend rooted at basePath.
func NewFilesystem(basePath string) *Filesystem {
	return &Filesystem{basePath: basePath}
}

func (f *Filesystem) ReadManifest(_ context.Context) (*manifest.Manifest, error) {
	file, err := os.Open(filepath.Join(f.basePath, "manifest.json"))
	if err != nil {
		return nil, fmt.Errorf("read manifest: %w", err)
	}
	defer func() { _ = file.Close() }()
	return manifest.Read(file)
}

func (f *Filesystem) ListFiles(_ context.Context, prefix string) ([]string, error) {
	dir := filepath.Join(f.basePath, filepath.FromSlash(prefix))
	var files []string

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(f.basePath, path)
		if err != nil {
			return err
		}
		files = append(files, filepath.ToSlash(rel))
		return nil
	})
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("list files %q: %w", prefix, err)
	}
	return files, nil
}

func (f *Filesystem) ReadFile(_ context.Context, path string) (io.ReadCloser, error) {
	full := filepath.Join(f.basePath, filepath.FromSlash(path))
	file, err := os.Open(full)
	if err != nil {
		return nil, fmt.Errorf("read file %q: %w", path, err)
	}
	return file, nil
}

func (f *Filesystem) WriteManifest(_ context.Context, m *manifest.Manifest) error {
	var buf bytes.Buffer
	if err := m.Write(&buf); err != nil {
		return fmt.Errorf("serialize manifest: %w", err)
	}
	path := filepath.Join(f.basePath, "manifest.json")
	return os.WriteFile(path, buf.Bytes(), 0644)
}

func (f *Filesystem) WriteFile(_ context.Context, path string, r io.Reader) error {
	full := filepath.Join(f.basePath, filepath.FromSlash(path))
	if err := os.MkdirAll(filepath.Dir(full), 0755); err != nil {
		return fmt.Errorf("create directory for %q: %w", path, err)
	}
	data, err := io.ReadAll(r)
	if err != nil {
		return fmt.Errorf("read content for %q: %w", path, err)
	}
	return os.WriteFile(full, data, 0644)
}

func (f *Filesystem) DeleteFile(_ context.Context, path string) error {
	full := filepath.Join(f.basePath, filepath.FromSlash(path))
	err := os.Remove(full)
	if err != nil && os.IsNotExist(err) {
		return nil
	}
	return err
}

// BasePath returns the root directory of this filesystem storage.
func (f *Filesystem) BasePath() string {
	return f.basePath
}

// Ensure Filesystem implements ReaderWriter.
var _ ReaderWriter = (*Filesystem)(nil)

// ParseStorageURL returns the appropriate backend for a given URL/path.
// For now, only filesystem paths are supported. S3/GCS added in a later release.
func ParseStorageURL(rawURL string) (ReaderWriter, error) {
	if strings.HasPrefix(rawURL, "s3://") {
		return nil, fmt.Errorf("S3 storage not yet implemented")
	}
	if strings.HasPrefix(rawURL, "gs://") {
		return nil, fmt.Errorf("GCS storage not yet implemented")
	}
	// Treat as filesystem path
	return NewFilesystem(rawURL), nil
}
