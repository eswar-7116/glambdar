package storage

import (
	"bytes"
	"context"
	"os"
	"sync"
	"testing"
)

func TestMockStorage_UploadDownload(t *testing.T) {
	m := NewMockStorage()
	ctx := context.Background()
	data := []byte("hello world")

	if err := m.Upload(ctx, "test-key", bytes.NewReader(data)); err != nil {
		t.Fatalf("Upload failed: %v", err)
	}

	rc, err := m.Download(ctx, "test-key")
	if err != nil {
		t.Fatalf("Download failed: %v", err)
	}
	defer rc.Close()

	var buf bytes.Buffer
	if _, err := buf.ReadFrom(rc); err != nil {
		t.Fatalf("failed to read downloaded data: %v", err)
	}
	if !bytes.Equal(buf.Bytes(), data) {
		t.Errorf("downloaded data mismatch: got %q, want %q", buf.String(), string(data))
	}
}

func TestMockStorage_DownloadMissing(t *testing.T) {
	m := NewMockStorage()
	ctx := context.Background()

	_, err := m.Download(ctx, "nonexistent")
	if err == nil {
		t.Error("expected error when downloading non-existent key")
	}
	if !os.IsNotExist(err) {
		t.Errorf("expected os.ErrNotExist, got: %v", err)
	}
}

func TestMockStorage_Delete(t *testing.T) {
	m := NewMockStorage()
	ctx := context.Background()

	if err := m.Upload(ctx, "delete-me", bytes.NewReader([]byte("data"))); err != nil {
		t.Fatalf("Upload failed: %v", err)
	}

	if err := m.Delete(ctx, "delete-me"); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	_, err := m.Download(ctx, "delete-me")
	if err == nil {
		t.Error("expected error after deleting key")
	}
}

func TestMockStorage_UploadNilReader(t *testing.T) {
	m := NewMockStorage()
	ctx := context.Background()

	if err := m.Upload(ctx, "nil-reader", nil); err != nil {
		t.Fatalf("Upload with nil reader failed: %v", err)
	}

	rc, err := m.Download(ctx, "nil-reader")
	if err != nil {
		t.Fatalf("Download failed: %v", err)
	}
	defer rc.Close()

	var buf bytes.Buffer
	buf.ReadFrom(rc)
	if buf.Len() != 0 {
		t.Errorf("expected empty data for nil reader upload, got %d bytes", buf.Len())
	}
}

func TestMockStorage_ConcurrentAccess(t *testing.T) {
	m := NewMockStorage()
	ctx := context.Background()

	var wg sync.WaitGroup
	for i := range 50 {
		wg.Add(3)
		key := "key-" + string(rune('a'+i%26))
		wg.Go(func() {
			defer wg.Done()
			_ = m.Upload(ctx, key, bytes.NewReader([]byte("data")))
		})
		wg.Go(func() {
			defer wg.Done()
			m.Download(ctx, key)
		})
		wg.Go(func() {
			defer wg.Done()
			_ = m.Delete(ctx, key)
		})
	}
	wg.Wait()
}
