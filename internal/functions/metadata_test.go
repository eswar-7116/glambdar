package functions_test

import (
	"testing"
	"time"

	"github.com/eswar-7116/glambdar/v3/internal/config"
	"github.com/eswar-7116/glambdar/v3/internal/functions"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func setupTestDB(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("failed to create memory db: %v", err)
	}
	db.AutoMigrate(&functions.Metadata{}, &functions.Log{})
	config.DB = db
}

func TestSaveAndLoadMetadata(t *testing.T) {
	setupTestDB(t)

	metadata := &functions.Metadata{
		Name:          "Test Function",
		CreatedAt:     time.Now().UTC(),
		LastInvokedAt: time.Now().UTC(),
		InvokeCount:   5,
	}

	err := functions.SaveMetadata(metadata)
	if err != nil {
		t.Fatalf("expected no error saving metadata, got %v", err)
	}

	loadedMetadata, err := functions.LoadMetadata(metadata.Name)
	if err != nil {
		t.Fatalf("expected no error loading metadata, got %v", err)
	}

	if loadedMetadata.Name != metadata.Name {
		t.Errorf("expected Name %s, got %s", metadata.Name, loadedMetadata.Name)
	}
	if loadedMetadata.InvokeCount != metadata.InvokeCount {
		t.Errorf("expected InvokeCount %d, got %d", metadata.InvokeCount, loadedMetadata.InvokeCount)
	}
	
	err = functions.DeleteMetadata(metadata.Name)
	if err != nil {
		t.Fatalf("expected no error deleting metadata, got %v", err)
	}
	
	_, err = functions.LoadMetadata(metadata.Name)
	if err == nil {
		t.Errorf("expected error loading deleted metadata, got nil")
	}
}

func TestLoadMetadataFileNotFound(t *testing.T) {
	setupTestDB(t)
	_, err := functions.LoadMetadata("non_existent_func")
	if err == nil {
		t.Fatal("expected error when loading from a non-existent file, got nil")
	}
}
