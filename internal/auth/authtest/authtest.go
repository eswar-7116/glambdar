package authtest

import (
	"fmt"
	"testing"
	"time"

	"github.com/eswar-7116/glambdar/v3/internal/auth"
	"github.com/eswar-7116/glambdar/v3/internal/config"
)

func SetupTestAuth(t *testing.T) string {
	t.Helper()
	config.DB.AutoMigrate(&auth.APIKey{}, &auth.AuditLog{})

	rawKey, err := auth.GenerateAPIKey(auth.RoleAdmin)
	if err != nil {
		t.Fatalf("failed to generate test admin key: %v", err)
	}

	apiKey := &auth.APIKey{
		KeyHash:   auth.HashKey(rawKey),
		KeyPrefix: auth.KeyPrefixFromRaw(rawKey),
		Name:      fmt.Sprintf("test-root-%d", time.Now().UnixNano()),
		Role:      auth.RoleAdmin,
		IsRoot:    true,
	}
	if err := config.DB.Create(apiKey).Error; err != nil {
		t.Fatalf("failed to create test admin key: %v", err)
	}
	return rawKey
}
