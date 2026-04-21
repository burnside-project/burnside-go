package storage

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"

	gcs "cloud.google.com/go/storage"
	"google.golang.org/api/iterator"

	"github.com/burnside-project/burnside-go/manifest"
)

// GCS implements ReaderWriter using Google Cloud Storage.
type GCS struct {
	client *gcs.Client
	bucket string
	prefix string
}

// NewGCS creates a GCS storage backend.
func NewGCS(client *gcs.Client, bucket, prefix string) *GCS {
	prefix = strings.TrimPrefix(prefix, "/")
	if prefix != "" && !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}
	return &GCS{client: client, bucket: bucket, prefix: prefix}
}

func (g *GCS) key(path string) string {
	return g.prefix + path
}

func (g *GCS) bkt() *gcs.BucketHandle {
	return g.client.Bucket(g.bucket)
}

func (g *GCS) ReadManifest(ctx context.Context) (*manifest.Manifest, error) {
	rc, err := g.ReadFile(ctx, "manifest.json")
	if err != nil {
		return nil, fmt.Errorf("read manifest: %w", err)
	}
	defer func() { _ = rc.Close() }()
	return manifest.Read(rc)
}

func (g *GCS) ListFiles(ctx context.Context, prefix string) ([]string, error) {
	fullPrefix := g.key(prefix)

	it := g.bkt().Objects(ctx, &gcs.Query{Prefix: fullPrefix})
	var files []string
	for {
		attrs, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("list gs://%s/%s: %w", g.bucket, fullPrefix, err)
		}
		rel := strings.TrimPrefix(attrs.Name, g.prefix)
		files = append(files, rel)
	}
	return files, nil
}

func (g *GCS) ReadFile(ctx context.Context, path string) (io.ReadCloser, error) {
	rc, err := g.bkt().Object(g.key(path)).NewReader(ctx)
	if err != nil {
		return nil, fmt.Errorf("read gs://%s/%s: %w", g.bucket, g.key(path), err)
	}
	return rc, nil
}

func (g *GCS) WriteManifest(ctx context.Context, m *manifest.Manifest) error {
	var buf bytes.Buffer
	if err := m.Write(&buf); err != nil {
		return fmt.Errorf("serialize manifest: %w", err)
	}
	return g.writeBytes(ctx, "manifest.json", buf.Bytes(), "application/json")
}

func (g *GCS) WriteFile(ctx context.Context, path string, r io.Reader) error {
	w := g.bkt().Object(g.key(path)).NewWriter(ctx)
	if _, err := io.Copy(w, r); err != nil {
		_ = w.Close()
		return fmt.Errorf("write gs://%s/%s: %w", g.bucket, g.key(path), err)
	}
	return w.Close()
}

func (g *GCS) DeleteFile(ctx context.Context, path string) error {
	err := g.bkt().Object(g.key(path)).Delete(ctx)
	if err == gcs.ErrObjectNotExist {
		return nil
	}
	return err
}

func (g *GCS) writeBytes(ctx context.Context, path string, data []byte, contentType string) error {
	w := g.bkt().Object(g.key(path)).NewWriter(ctx)
	w.ContentType = contentType
	if _, err := w.Write(data); err != nil {
		_ = w.Close()
		return fmt.Errorf("write gs://%s/%s: %w", g.bucket, g.key(path), err)
	}
	return w.Close()
}

// Ensure GCS implements ReaderWriter.
var _ ReaderWriter = (*GCS)(nil)
