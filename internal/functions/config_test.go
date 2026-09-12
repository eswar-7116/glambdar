package functions_test

import (
	"testing"

	"github.com/eswar-7116/glambdar/v3/internal/config"
	"github.com/eswar-7116/glambdar/v3/internal/functions"
	"github.com/eswar-7116/glambdar/v3/internal/testutil"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestUpdateRateLimit(t *testing.T) {
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

	funcName := "test-func"
	// Create initial metadata
	config.DB.Create(&functions.Metadata{
		Name:      funcName,
		RateLimit: 10,
	})

	// Test case: Happy path
	err = functions.UpdateRateLimit(funcName, 20)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	md, _ := functions.LoadMetadata(funcName)
	if md.RateLimit != 20 {
		t.Errorf("expected rate limit 20, got %d", md.RateLimit)
	}

	// Test case: Non-existent function
	err = functions.UpdateRateLimit("missing-func", 30)
	if err == nil {
		t.Error("expected error for non-existent function, got nil")
	}
}
