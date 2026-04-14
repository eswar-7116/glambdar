package api

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/eswar-7116/glambdar/internal/config"
	"github.com/eswar-7116/glambdar/internal/functions"
	"github.com/gin-gonic/gin"
)

func TestDeleteHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tempDir, err := os.MkdirTemp("", "glambdar-api-delete-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	config.InitPathsWithBase(tempDir)
	config.DB.AutoMigrate(&functions.Metadata{}, &functions.Log{})

	// Create a dummy function directory and metadata
	funcName := "testfunc"
	funcDir := filepath.Join(tempDir, "functions", funcName)
	os.MkdirAll(funcDir, 0755)
	functions.SaveMetadata(&functions.Metadata{Name: funcName})

	// Create a mock log
	functions.SaveLog(&functions.Log{
		FuncName: funcName,
		Stdout:   "hello",
	})

	router := Router()

	// Verify log exists
	logs, _ := functions.GetLogsByFunction(funcName)
	if len(logs) == 0 {
		t.Fatalf("expected log to exist before deletion")
	}

	// Test case: Delete existing function
	req, _ := http.NewRequest("DELETE", "/del/"+funcName, nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	if _, err := os.Stat(funcDir); !os.IsNotExist(err) {
		t.Errorf("expected function directory to be deleted, but it still exists")
	}

	// Verify log is deleted
	logs, _ = functions.GetLogsByFunction(funcName)
	if len(logs) != 0 {
		t.Errorf("expected logs to be deleted, but found %d logs", len(logs))
	}

	// Test case: Delete non-existing function
	req, _ = http.NewRequest("DELETE", "/del/"+funcName, nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", w.Code)
	}
}
