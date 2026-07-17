package api

import (
	"bytes"
	"mime/multipart"
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

func TestDeployHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tempDir, err := os.MkdirTemp("", "glambdar-api-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	config.InitPathsWithBase(tempDir)
	config.DB.AutoMigrate(&functions.Metadata{}, &functions.Log{})
	adminKey := authtest.SetupTestAuth(t)

	router := Router()

	// Test case: Missing file
	req, _ := http.NewRequest("POST", "/deploy", nil)
	req.Header.Set("X-API-Key", adminKey)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}

	// Test case: Valid deployment
	// We need a dummy zip file.
	zipPath := filepath.Join("..", "..", "test_data", "zip", "valid.zip")
	zipContent, err := os.ReadFile(zipPath)
	if err != nil {
		t.Fatalf("failed to read test zip file: %v", err)
	}

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", "valid.zip")
	if err != nil {
		t.Fatalf("failed to create form file: %v", err)
	}
	part.Write(zipContent)
	writer.Close()

	req, _ = http.NewRequest("POST", "/deploy", body)
	req.Header.Set("X-API-Key", adminKey)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("expected status 201, got %d. Body: %s", w.Code, w.Body.String())
	}

	// Test case: Already exists
	body2 := &bytes.Buffer{}
	writer2 := multipart.NewWriter(body2)
	part2, err := writer2.CreateFormFile("file", "valid.zip")
	if err != nil {
		t.Fatalf("failed to create form file: %v", err)
	}
	part2.Write(zipContent)
	writer2.Close()

	req, _ = http.NewRequest("POST", "/deploy", body2)
	req.Header.Set("X-API-Key", adminKey)
	req.Header.Set("Content-Type", writer2.FormDataContentType())
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400 for existing function, got %d", w.Code)
	}
}
