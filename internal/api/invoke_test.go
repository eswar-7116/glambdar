package api

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/eswar-7116/glambdar/internal/config"
	"github.com/gin-gonic/gin"
)

func TestInvokeHandler_NotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tempDir, err := os.MkdirTemp("", "glambdar-api-invoke-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	config.InitPathsWithBase(tempDir)

	router := Router()

	// Test case: Function not found
	req, _ := http.NewRequest("POST", "/invoke/missing", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", w.Code)
	}
}
