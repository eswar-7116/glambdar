package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/eswar-7116/glambdar/internal/config"
	"github.com/eswar-7116/glambdar/internal/functions"
	"github.com/gin-gonic/gin"
)

func TestLogsHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tempDir, err := os.MkdirTemp("", "glambdar-api-logs-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	config.InitPathsWithBase(tempDir)
	config.DB.AutoMigrate(&functions.Metadata{}, &functions.Log{})

	funcName := "logtest"
	functions.SaveMetadata(&functions.Metadata{Name: funcName})
	
	// Insert mocks
	functions.SaveLog(&functions.Log{FuncName: funcName, Stdout: "out1", Stderr: "err1"})
	functions.SaveLog(&functions.Log{FuncName: funcName, Stdout: "out2", Stderr: "err2"})

	router := Router()

	// Test case: Success
	req, _ := http.NewRequest("GET", "/logs/"+funcName, nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var logs []functions.Log
	json.Unmarshal(w.Body.Bytes(), &logs)
	if len(logs) != 2 {
		t.Errorf("expected 2 logs, got %d", len(logs))
	}

	// Test case: Not found
	req, _ = http.NewRequest("GET", "/logs/missing", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", w.Code)
	}
}
