package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/eswar-7116/glambdar/internal/config"
	"github.com/eswar-7116/glambdar/internal/functions"
	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

func TestInvokeHandler_NotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tempDir, err := os.MkdirTemp("", "glambdar-api-invoke-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	config.InitPathsWithBase(tempDir)
	config.DB.AutoMigrate(&functions.Metadata{}, &functions.Log{})

	router := Router()

	// Test case: Function not found
	req, _ := http.NewRequest("POST", "/invoke/missing", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", w.Code)
	}
}

func TestInvokeHandler_RateLimited(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tempDir, err := os.MkdirTemp("", "glambdar-api-ratelimit-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	config.InitPathsWithBase(tempDir)
	config.DB.AutoMigrate(&functions.Metadata{}, &functions.Log{})

	// Create function directory so it passes the existence check
	funcName := "limited-func"
	os.MkdirAll(filepath.Join(config.FunctionsDir, funcName), 0755)

	// Create metadata in DB
	config.DB.Create(&functions.Metadata{
		Name:      funcName,
		RateLimit: 1, // Will override later or just use this
	})

	// Mock a rate-limited pool
	p := config.PoolManager.GetOrCreate(funcName, 1, 1)
	p.Limiter = rate.NewLimiter(rate.Limit(0), 0) // Never allow

	router := Router()

	req, _ := http.NewRequest("POST", "/invoke/"+funcName, strings.NewReader(`{}`))
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusTooManyRequests {
		t.Errorf("expected status 429, got %d", w.Code)
	}

	var resp map[string]string
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["error"] != "Rate limit exceeded. Please try again later." {
		t.Errorf("unexpected error message: %s", resp["error"])
	}
}
