package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"time"

	"github.com/eswar-7116/glambdar/v3/internal/config"
)

// Role defines the access level
type Role string

const (
	RoleAdmin    Role = "admin"
	RoleDeployer Role = "deployer"
	RoleInvoker  Role = "invoker"
	RoleViewer   Role = "viewer"
)

var validRoles = map[Role]bool{
	RoleAdmin:    true,
	RoleDeployer: true,
	RoleInvoker:  true,
	RoleViewer:   true,
}

// APIKey represents an auth key stored in the DB
type APIKey struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	KeyHash   string    `json:"-" gorm:"uniqueIndex;size:64;not null"`
	KeyPrefix string    `json:"keyPrefix" gorm:"size:16;not null"`
	Name      string    `json:"name" gorm:"uniqueIndex;size:100;not null"`
	Role      Role      `json:"role" gorm:"size:20;not null"`
	IsRoot    bool      `json:"isRoot" gorm:"default:false"`
	CreatedAt time.Time `json:"createdAt"`
}

// AuditLog tracks authenticated actions
type AuditLog struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	Timestamp time.Time `json:"timestamp" gorm:"index"`
	ActorName string    `json:"actorName" gorm:"size:100"`
	ActorRole string    `json:"actorRole" gorm:"size:20"`
	Action    string    `json:"action" gorm:"size:50"`
	Method    string    `json:"method" gorm:"size:10"`
	Path      string    `json:"path"`
	Status    int       `json:"status"`
}

var rolePrefix = map[Role]string{
	RoleAdmin:    "glmbd_ak_",
	RoleDeployer: "glmbd_dk_",
	RoleInvoker:  "glmbd_ik_",
	RoleViewer:   "glmbd_vk_",
}

// GenerateAPIKey creates a random API key
func GenerateAPIKey(role Role) (string, error) {
	prefix, ok := rolePrefix[role]
	if !ok {
		return "", fmt.Errorf("unknown role: %s", role)
	}

	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("failed to generate random bytes: %w", err)
	}

	return prefix + hex.EncodeToString(b), nil
}

// HashKey computes SHA-256 hash of key
func HashKey(rawKey string) string {
	hash := sha256.Sum256([]byte(rawKey))
	return hex.EncodeToString(hash[:])
}

// KeyPrefixFromRaw gets the prefix
func KeyPrefixFromRaw(rawKey string) string {
	if len(rawKey) < 16 {
		return rawKey
	}
	return rawKey[:16]
}

// ValidateRole ensures role is valid
func ValidateRole(s string) (Role, error) {
	r := Role(s)
	if !validRoles[r] {
		return "", fmt.Errorf("invalid role: %q", s)
	}
	return r, nil
}

// BootstrapRootKey creates root key on first boot
func BootstrapRootKey() error {
	var count int64
	if err := config.DB.Model(&APIKey{}).Where("is_root = ?", true).Count(&count).Error; err != nil {
		return fmt.Errorf("failed to check root key: %w", err)
	}

	if count > 0 {
		return nil
	}

	rawKey, err := GenerateAPIKey(RoleAdmin)
	if err != nil {
		return fmt.Errorf("failed to generate root key: %w", err)
	}

	apiKey := &APIKey{
		KeyHash:   HashKey(rawKey),
		KeyPrefix: KeyPrefixFromRaw(rawKey),
		Name:      "root",
		Role:      RoleAdmin,
		IsRoot:    true,
	}
	if err := config.DB.Create(apiKey).Error; err != nil {
		return fmt.Errorf("failed to store root key: %w", err)
	}

	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintf(os.Stderr, "Admin API key: %s\n", rawKey)
	fmt.Fprintln(os.Stderr, "Save this key securely. It will NOT be shown again.")
	fmt.Fprintln(os.Stderr, "")

	return nil
}

// ResetRootKey resets root key via CLI
func ResetRootKey() error {
	var rootKey APIKey
	if err := config.DB.Where("is_root = ?", true).First(&rootKey).Error; err != nil {
		return fmt.Errorf("no root key exists — start glambdar first")
	}

	rawKey, err := GenerateAPIKey(RoleAdmin)
	if err != nil {
		return fmt.Errorf("failed to generate root key: %w", err)
	}

	rootKey.KeyHash = HashKey(rawKey)
	rootKey.KeyPrefix = KeyPrefixFromRaw(rawKey)
	if err := config.DB.Save(&rootKey).Error; err != nil {
		return fmt.Errorf("failed to update root key: %w", err)
	}

	fmt.Printf("\nNew admin API key: %s\n", rawKey)
	fmt.Println("Save this key securely. It will NOT be shown again.")
	fmt.Println("The previous admin key has been invalidated.")

	return nil
}
