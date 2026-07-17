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

func setupLogsTestDB(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("failed to create memory db: %v", err)
	}
	db.AutoMigrate(&functions.Metadata{}, &functions.Log{})
	config.DB = db
}

func TestSaveAndGetLogs(t *testing.T) {
	setupLogsTestDB(t)

	funcName := "test-func"
	log1 := &functions.Log{
		FuncName:  funcName,
		InvokedAt: time.Now().Add(-10 * time.Minute),
		Stdout:    "output 1",
		Stderr:    "",
	}
	log2 := &functions.Log{
		FuncName:  funcName,
		InvokedAt: time.Now().Add(-5 * time.Minute),
		Stdout:    "output 2",
		Stderr:    "error 2",
	}

	err := functions.SaveLog(log1)
	if err != nil {
		t.Fatalf("SaveLog(log1) failed: %v", err)
	}

	err = functions.SaveLog(log2)
	if err != nil {
		t.Fatalf("SaveLog(log2) failed: %v", err)
	}

	logs, err := functions.GetLogsByFunction(funcName)
	if err != nil {
		t.Fatalf("GetLogsByFunction failed: %v", err)
	}

	if len(logs) != 2 {
		t.Errorf("expected 2 logs, got %d", len(logs))
	}

	// Should be ordered by InvokedAt DESC
	if logs[0].Stdout != "output 2" {
		t.Errorf("expected first log to be 'output 2', got '%s'", logs[0].Stdout)
	}
	if logs[1].Stdout != "output 1" {
		t.Errorf("expected second log to be 'output 1', got '%s'", logs[1].Stdout)
	}
}

func TestDeleteLogs(t *testing.T) {
	setupLogsTestDB(t)

	funcName := "delete-test-func"
	functions.SaveLog(&functions.Log{
		FuncName:  funcName,
		InvokedAt: time.Now(),
		Stdout:    "some output",
	})

	err := functions.DeleteLogsByFunction(funcName)
	if err != nil {
		t.Fatalf("DeleteLogsByFunction failed: %v", err)
	}

	logs, err := functions.GetLogsByFunction(funcName)
	if err != nil {
		t.Fatalf("GetLogsByFunction failed: %v", err)
	}

	if len(logs) != 0 {
		t.Errorf("expected 0 logs after deletion, got %d", len(logs))
	}
}

func TestGetLogsEmpty(t *testing.T) {
	setupLogsTestDB(t)

	logs, err := functions.GetLogsByFunction("non-existent")
	if err != nil {
		t.Fatalf("GetLogsByFunction failed: %v", err)
	}

	if len(logs) != 0 {
		t.Errorf("expected 0 logs for non-existent function, got %d", len(logs))
	}
}
