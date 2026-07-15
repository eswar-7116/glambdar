package authtest

import (
	"testing"
	"github.com/eswar-7116/glambdar/internal/auth"
	"github.com/eswar-7116/glambdar/internal/config"
)

func SetupTestAuth(t *testing.T) string {
	t.Helper()
	config.DB.AutoMigrate(&auth.APIKey{}, &auth.AuditLog{})

	rawKey, err := auth.GenerateAPIKey(auth.RoleAdmin)
	if err != nil {
		t.Fatalf("failed to generate test admin key: %v", err)
	}

	apiKey := &auth.APIKey{
		KeyPrefix: auth.KeyPrefixFromRaw(rawKey),
		Name:      "test-root",
		Role:      auth.RoleAdmin,
		IsRoot:    true,
	}
	// Use reflection or just set hash, but wait KeyHash is exported!
	apiKey.KeyHash = auth.HashKey(rawKey)
	
	if err := config.DB.Create(apiKey).Error; err != nil {
		t.Fatalf("failed to create test admin key: %v", err)
	}
	return rawKey
}
