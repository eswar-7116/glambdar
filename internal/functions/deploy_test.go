package functions_test

import (
	"os"
	"path/filepath"
	"testing"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/eswar-7116/glambdar/internal/config"
	"github.com/eswar-7116/glambdar/internal/functions"
)

var validZipFile = filepath.Join("..", "..", "test_data", "zip", "valid.zip")

func TestDeploy_CreatesFunctionAndMetadata(t *testing.T) {
	tmp := t.TempDir()
	config.FunctionsDir = tmp

	// setup test db
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("failed to create memory db: %v", err)
	}
	db.AutoMigrate(&functions.Metadata{}, &functions.Log{})
	config.DB = db

	t.Log("Deploying...")
	err = functions.Deploy(validZipFile, "testFunc")
	if err != nil {
		t.Fatalf("deploy failed: %v", err)
	}

	funcDir := filepath.Join(config.FunctionsDir, "testFunc")
	if _, err := os.Stat(funcDir); err != nil {
		t.Fatalf("function directory not created")
	}

	md, err := functions.LoadMetadata("testFunc")
	if err != nil {
		t.Fatalf("metadata not created")
	}

	if md.Name != "testFunc" {
		t.Fatalf("metadata name mismatch")
	}
	if md.InvokeCount != 0 {
		t.Fatalf("invoke count should be zero")
	}
}
