package storage

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/dataalgebra-engineering/burnside-project-burnside-go/manifest"
)

func TestFilesystem_WriteAndReadManifest(t *testing.T) {
	dir := t.TempDir()
	fs := NewFilesystem(dir)
	ctx := context.Background()

	m := &manifest.Manifest{
		Version: 1,
		Source: manifest.Source{
			Name:    "test",
			InitLSN: "0/1000",
		},
		Tables: map[string]manifest.Table{
			"public.orders": {
				Status:           "active",
				PrimaryKey:       []string{"id"},
				BaseEpoch:        0,
				LatestDeltaEpoch: 3,
				Path:             "public.orders",
			},
		},
		Schemas:   map[string]manifest.Schema{},
		UpdatedAt: time.Now().UTC().Truncate(time.Second),
	}

	if err := fs.WriteManifest(ctx, m); err != nil {
		t.Fatalf("WriteManifest: %v", err)
	}

	got, err := fs.ReadManifest(ctx)
	if err != nil {
		t.Fatalf("ReadManifest: %v", err)
	}
	if got.Source.InitLSN != "0/1000" {
		t.Errorf("InitLSN = %q", got.Source.InitLSN)
	}
	if got.Tables["public.orders"].LatestDeltaEpoch != 3 {
		t.Error("table data mismatch")
	}
}

func TestFilesystem_WriteAndReadFile(t *testing.T) {
	dir := t.TempDir()
	fs := NewFilesystem(dir)
	ctx := context.Background()

	content := "hello parquet bytes"
	err := fs.WriteFile(ctx, "public.orders/base/base_000000.parquet", strings.NewReader(content))
	if err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	// Verify file exists on disk
	full := filepath.Join(dir, "public.orders", "base", "base_000000.parquet")
	if _, err := os.Stat(full); err != nil {
		t.Fatalf("file not on disk: %v", err)
	}

	// Read it back
	rc, err := fs.ReadFile(ctx, "public.orders/base/base_000000.parquet")
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	defer func() { _ = rc.Close() }()
	buf := make([]byte, 100)
	n, _ := rc.Read(buf)
	if string(buf[:n]) != content {
		t.Errorf("content = %q", string(buf[:n]))
	}
}

func TestFilesystem_ListFiles(t *testing.T) {
	dir := t.TempDir()
	fs := NewFilesystem(dir)
	ctx := context.Background()

	// Create some files
	_ = fs.WriteFile(ctx, "public.orders/deltas/epoch=000001.parquet", strings.NewReader("a"))
	_ = fs.WriteFile(ctx, "public.orders/deltas/epoch=000002.parquet", strings.NewReader("b"))
	_ = fs.WriteFile(ctx, "public.orders/base/base_000000.parquet", strings.NewReader("c"))

	files, err := fs.ListFiles(ctx, "public.orders/deltas/")
	if err != nil {
		t.Fatalf("ListFiles: %v", err)
	}
	if len(files) != 2 {
		t.Errorf("listed %d files, want 2", len(files))
	}
	for _, f := range files {
		if !strings.Contains(f, "epoch=") {
			t.Errorf("unexpected file: %s", f)
		}
	}
}

func TestFilesystem_ListFiles_NonexistentDir(t *testing.T) {
	dir := t.TempDir()
	fs := NewFilesystem(dir)
	ctx := context.Background()

	files, err := fs.ListFiles(ctx, "nonexistent/path/")
	if err != nil {
		t.Fatalf("ListFiles should not error on missing dir: %v", err)
	}
	if len(files) != 0 {
		t.Errorf("expected empty, got %d", len(files))
	}
}

func TestFilesystem_DeleteFile(t *testing.T) {
	dir := t.TempDir()
	fs := NewFilesystem(dir)
	ctx := context.Background()

	_ = fs.WriteFile(ctx, "temp.parquet", strings.NewReader("data"))

	if err := fs.DeleteFile(ctx, "temp.parquet"); err != nil {
		t.Fatalf("DeleteFile: %v", err)
	}
	// Should not error on already deleted
	if err := fs.DeleteFile(ctx, "temp.parquet"); err != nil {
		t.Fatalf("DeleteFile (second): %v", err)
	}
}

func TestParseStorageURL(t *testing.T) {
	// Filesystem path
	rw, err := ParseStorageURL("/tmp/cdc-output")
	if err != nil {
		t.Fatalf("ParseStorageURL: %v", err)
	}
	fs, ok := rw.(*Filesystem)
	if !ok {
		t.Fatal("expected Filesystem")
	}
	if fs.BasePath() != "/tmp/cdc-output" {
		t.Errorf("BasePath = %q", fs.BasePath())
	}

	// S3 not implemented
	_, err = ParseStorageURL("s3://bucket/prefix")
	if err == nil {
		t.Error("expected error for s3://")
	}
}
