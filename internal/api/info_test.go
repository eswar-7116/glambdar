package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/eswar-7116/glambdar/v3/internal/auth/authtest"
	"github.com/eswar-7116/glambdar/v3/internal/config"
	"github.com/eswar-7116/glambdar/v3/internal/functions"
	"github.com/gin-gonic/gin"
)

func TestInfoHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tempDir, err := os.MkdirTemp("", "glambdar-api-info-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	config.InitPathsWithBase(tempDir)
	config.DB.AutoMigrate(&functions.Metadata{}, &functions.Log{})
	adminKey := authtest.SetupTestAuth(t)

	// Create a dummy function directory with metadata
	funcName := "testfunc"
	funcDir := filepath.Join(tempDir, "functions", funcName)
	os.MkdirAll(funcDir, 0755)
	
	md := &functions.Metadata{
		Name: funcName,
	}
	functions.SaveMetadata(md)

	router := Router()

	// Test case: /info (list all)
	req, _ := http.NewRequest("GET", "/info", nil)
	req.Header.Set("X-API-Key", adminKey)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var res map[string]any
	json.Unmarshal(w.Body.Bytes(), &res)
	if res["count"].(float64) != 1 {
		t.Errorf("expected count 1, got %v", res["count"])
	}

	// Test case: /info/:name (get specific)
	req, _ = http.NewRequest("GET", "/info/"+funcName, nil)
	req.Header.Set("X-API-Key", adminKey)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var resMd functions.Metadata
	json.Unmarshal(w.Body.Bytes(), &resMd)
	if resMd.Name != funcName {
		t.Errorf("expected name %s, got %s", funcName, resMd.Name)
	}

	// Test case: /info/:name (not found)
	req, _ = http.NewRequest("GET", "/info/missing", nil)
	req.Header.Set("X-API-Key", adminKey)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", w.Code)
	}
}
