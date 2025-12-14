package functions_test

import (
	"testing"
	"time"

	"github.com/eswar-7116/glambdar/internal/functions"
)

func TestSaveAndLoadMetadata(t *testing.T) {
	funcDir := t.TempDir()

	metadata := &functions.Metadata{
		Name:          "Test Function",
		CreatedAt:     time.Now(),
		LastInvokedAt: time.Now(),
		InvokeCount:   5,
	}

	err := functions.SaveMetadata(funcDir, metadata)
	if err != nil {
		t.Fatalf("expected no error saving metadata, got %v", err)
	}

	loadedMetadata, err := functions.LoadMetadata(funcDir)
	if err != nil {
		t.Fatalf("expected no error loading metadata, got %v", err)
	}

	if loadedMetadata.Name != metadata.Name {
		t.Errorf("expected Name %s, got %s", metadata.Name, loadedMetadata.Name)
	}
	if !loadedMetadata.CreatedAt.Equal(metadata.CreatedAt) {
		t.Errorf("expected CreatedAt %v, got %v", metadata.CreatedAt, loadedMetadata.CreatedAt)
	}
	if !loadedMetadata.LastInvokedAt.Equal(metadata.LastInvokedAt) {
		t.Errorf("expected LastInvokedAt %v, got %v", metadata.LastInvokedAt, loadedMetadata.LastInvokedAt)
	}
	if loadedMetadata.InvokeCount != metadata.InvokeCount {
		t.Errorf("expected InvokeCount %d, got %d", metadata.InvokeCount, loadedMetadata.InvokeCount)
	}
}

func TestLoadMetadataFileNotFound(t *testing.T) {
	funcDir := t.TempDir()
	_, err := functions.LoadMetadata(funcDir)
	if err == nil {
		t.Fatal("expected error when loading from a non-existent file, got nil")
	}
}

func TestLoadMetadataInvalidData(t *testing.T) {
	funcDir := "../test_data/meta"

	_, err := functions.LoadMetadata(funcDir)
	if err == nil {
		t.Fatal("expected error when loading invalid metadata, got nil")
	}
}
