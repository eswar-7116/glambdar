package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/eswar-7116/glambdar/internal/config"
	"github.com/eswar-7116/glambdar/internal/functions"
	"github.com/gin-gonic/gin"
)

func TestConfigHandler_UpdateRateLimit(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tempDir, err := os.MkdirTemp("", "glambdar-api-config-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	config.InitPathsWithBase(tempDir)
	config.DB.AutoMigrate(&functions.Metadata{}, &functions.Log{})

	funcName := "config-func"
	os.MkdirAll(filepath.Join(config.FunctionsDir, funcName), 0755)

	// Create initial metadata
	config.DB.Create(&functions.Metadata{
		Name:      funcName,
		RateLimit: 10,
	})

	router := Router()

	// Update rate limit to 5
	reqBody, _ := json.Marshal(map[string]int{"rateLimit": 5})
	req, _ := http.NewRequest("POST", "/config/"+funcName, bytes.NewBuffer(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d. Body: %s", w.Code, w.Body.String())
	}

	// Verify DB update
	md, _ := functions.LoadMetadata(funcName)
	if md.RateLimit != 5 {
		t.Errorf("expected rate limit 5 in DB, got %d", md.RateLimit)
	}

	// Test case: Non-existent function
	req, _ = http.NewRequest("POST", "/config/missing-func", bytes.NewBuffer(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected status 500 for missing function, got %d", w.Code)
	}

	// Test case: Invalid JSON
	req, _ = http.NewRequest("POST", "/config/"+funcName, bytes.NewBufferString("invalid json"))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400 for invalid json, got %d", w.Code)
	}
}
