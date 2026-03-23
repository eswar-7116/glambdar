package api

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/eswar-7116/glambdar/internal/config"
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

	// Create a dummy function directory
	funcName := "testfunc"
	funcDir := filepath.Join(tempDir, "functions", funcName)
	os.MkdirAll(funcDir, 0755)

	router := Router()

	// Test case: Delete existing function
	req, _ := http.NewRequest("DELETE", "/del/"+funcName, nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("expected status 204, got %d", w.Code)
	}

	if _, err := os.Stat(funcDir); !os.IsNotExist(err) {
		t.Errorf("expected function directory to be deleted, but it still exists")
	}

	// Test case: Delete non-existing function
	req, _ = http.NewRequest("DELETE", "/del/"+funcName, nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", w.Code)
	}
}
