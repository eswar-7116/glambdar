package auth

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/eswar-7116/glambdar/v3/internal/config"
	"github.com/gin-gonic/gin"
)

// SetupTestAuth bootstraps a test DB with an admin key
func SetupTestAuth(t *testing.T) string {
	t.Helper()
	config.DB.AutoMigrate(&APIKey{}, &AuditLog{})

	rawKey, err := GenerateAPIKey(RoleAdmin)
	if err != nil {
		t.Fatalf("failed to generate test admin key: %v", err)
	}

	apiKey := &APIKey{
		KeyHash:   HashKey(rawKey),
		KeyPrefix: KeyPrefixFromRaw(rawKey),
		Name:      "test-root",
		Role:      RoleAdmin,
		IsRoot:    true,
	}
	if err := config.DB.Create(apiKey).Error; err != nil {
		t.Fatalf("failed to create test admin key: %v", err)
	}
	return rawKey
}

func setupIntegrationTest(t *testing.T) (string, *gin.Engine, func()) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	tempDir, err := os.MkdirTemp("", "glambdar-auth-integration-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	config.InitPathsWithBase(tempDir)
	config.DB.AutoMigrate(&APIKey{}, &AuditLog{})
	adminKey := SetupTestAuth(t)

	router := gin.Default()
	router.Use(AuthMiddleware())
	RegisterRoutes(router)
	
	// Add mock routes for testing middleware
	router.GET("/info", func(c *gin.Context) { c.JSON(200, gin.H{"status": "ok"}) })
	router.POST("/deploy", func(c *gin.Context) { c.JSON(201, gin.H{"status": "ok"}) })

	return adminKey, router, func() { os.RemoveAll(tempDir) }
}

func TestGenerateAPIKey(t *testing.T) {
	tests := []struct {
		role   Role
		prefix string
	}{
		{RoleAdmin, "glmbd_ak_"},
		{RoleDeployer, "glmbd_dk_"},
		{RoleInvoker, "glmbd_ik_"},
		{RoleViewer, "glmbd_vk_"},
	}

	for _, tt := range tests {
		key, err := GenerateAPIKey(tt.role)
		if err != nil {
			t.Fatalf("GenerateAPIKey error: %v", err)
		}
		if !strings.HasPrefix(key, tt.prefix) {
			t.Errorf("expected prefix %q, got key %q", tt.prefix, key)
		}
		if len(key) != 73 {
			t.Errorf("expected key length 73, got %d", len(key))
		}
	}
}

func TestHashKey(t *testing.T) {
	key := "glmbd_ak_test1234567890abcdef"
	h1 := HashKey(key)
	h2 := HashKey(key)
	if h1 != h2 {
		t.Error("HashKey should be deterministic")
	}
	h3 := HashKey(key + "x")
	if h1 == h3 {
		t.Error("different keys should produce different hashes")
	}
}

func TestValidateRole(t *testing.T) {
	for _, r := range []string{"admin", "deployer", "invoker", "viewer"} {
		if _, err := ValidateRole(r); err != nil {
			t.Errorf("ValidateRole unexpected error: %v", err)
		}
	}
	if _, err := ValidateRole("superadmin"); err == nil {
		t.Error("ValidateRole should reject invalid roles")
	}
}

func TestHasPermission_AdminFullAccess(t *testing.T) {
	cases := []struct{ action, method string }{
		{"deploy", "POST"},
		{"invoke", "GET"}, {"invoke", "POST"},
		{"auth", "GET"}, {"auth", "POST"},
	}
	for _, tt := range cases {
		if !HasPermission(RoleAdmin, tt.action, tt.method) {
			t.Errorf("admin should be allowed %s %s", tt.method, tt.action)
		}
	}
}

func TestMiddleware_MissingKey(t *testing.T) {
	_, router, cleanup := setupIntegrationTest(t)
	defer cleanup()

	req, _ := http.NewRequest("GET", "/info", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestMiddleware_InvalidKey(t *testing.T) {
	_, router, cleanup := setupIntegrationTest(t)
	defer cleanup()

	req, _ := http.NewRequest("GET", "/info", nil)
	req.Header.Set("X-API-Key", "glmbd_ak_fake")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestMiddleware_InsufficientRole(t *testing.T) {
	adminKey, router, cleanup := setupIntegrationTest(t)
	defer cleanup()

	body, _ := json.Marshal(map[string]string{"name": "viewer-user", "role": "viewer"})
	req, _ := http.NewRequest("POST", "/auth/keys", bytes.NewBuffer(body))
	req.Header.Set("X-API-Key", adminKey)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	var resp createKeyResponse
	json.Unmarshal(w.Body.Bytes(), &resp)

	req, _ = http.NewRequest("POST", "/deploy", nil)
	req.Header.Set("X-API-Key", resp.APIKey)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403, got %d", w.Code)
	}
}

func TestCreateKey(t *testing.T) {
	adminKey, router, cleanup := setupIntegrationTest(t)
	defer cleanup()

	body, _ := json.Marshal(map[string]string{"name": "ci-bot", "role": "deployer"})
	req, _ := http.NewRequest("POST", "/auth/keys", bytes.NewBuffer(body))
	req.Header.Set("X-API-Key", adminKey)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", w.Code)
	}
}

func TestDeleteRootKey_Blocked(t *testing.T) {
	adminKey, router, cleanup := setupIntegrationTest(t)
	defer cleanup()

	var rootKey APIKey
	config.DB.Where("is_root = ?", true).First(&rootKey)

	req, _ := http.NewRequest("DELETE", fmt.Sprintf("/auth/keys/%d", rootKey.ID), nil)
	req.Header.Set("X-API-Key", adminKey)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestBootstrapRootKey_Idempotent(t *testing.T) {
	tempDir, _ := os.MkdirTemp("", "glambdar-auth-bootstrap-*")
	defer os.RemoveAll(tempDir)

	config.InitPathsWithBase(tempDir)
	config.DB.AutoMigrate(&APIKey{})

	if err := BootstrapRootKey(); err != nil {
		t.Fatalf("first BootstrapRootKey failed: %v", err)
	}

	if err := BootstrapRootKey(); err != nil {
		t.Fatalf("second BootstrapRootKey failed: %v", err)
	}

	var count int64
	config.DB.Model(&APIKey{}).Where("is_root = ?", true).Count(&count)
	if count != 1 {
		t.Fatalf("expected 1 root key, got %d", count)
	}
}
