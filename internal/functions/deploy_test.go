package functions_test

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/eswar-7116/glambdar/v3/internal/config"
	"github.com/eswar-7116/glambdar/v3/internal/functions"
	"github.com/eswar-7116/glambdar/v3/internal/storage"
	"github.com/eswar-7116/glambdar/v3/internal/testutil"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var validZipFile = filepath.Join("..", "..", "test_data", "zip", "valid.zip")

func TestDeploy_CreatesFunctionAndMetadata(t *testing.T) {
	tmp := t.TempDir()
	config.FunctionsDir = tmp
	config.StorageClient = storage.NewMockStorage()

	// setup test db
	dsn := testutil.RequireTestDSN(t)
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("failed to create memory db: %v", err)
	}
	db.AutoMigrate(&functions.Metadata{}, &functions.Log{})
	config.DB = db
	testutil.ResetTestDB(t)

	t.Log("Deploying...")
	zipBytes, err := os.ReadFile(validZipFile)
	if err != nil {
		t.Fatalf("failed to read test zip file: %v", err)
	}
	err = functions.Deploy(context.Background(), "testFunc", bytes.NewReader(zipBytes), 0)
	if err != nil {
		t.Fatalf("deploy failed: %v", err)
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
