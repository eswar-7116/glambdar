package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestDBTypeJSON(t *testing.T) {
	tests := []struct {
		name     string
		dbType   DBType
		expected string
	}{
		{"sqlite", DBTypeSQLite, `"sqlite"`},
		{"postgres", DBTypePostgres, `"postgres"`},
		{"mysql", DBTypeMySQL, `"mysql"`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b, err := json.Marshal(tt.dbType)
			if err != nil {
				t.Fatalf("Failed to marshal: %v", err)
			}
			if string(b) != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, string(b))
			}

			var dt DBType
			if err := json.Unmarshal(b, &dt); err != nil {
				t.Fatalf("Failed to unmarshal: %v", err)
			}
			if dt != tt.dbType {
				t.Errorf("Expected %v, got %v", tt.dbType, dt)
			}
		})
	}
}

func TestDBConfigLoadSave(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "glambdar-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	originalConfigDir := ConfigDir
	ConfigDir = tempDir
	defer func() { ConfigDir = originalConfigDir }()

	t.Run("DefaultConfigCreation", func(t *testing.T) {
		cfg, err := LoadDBConfig()
		if err != nil {
			t.Fatalf("LoadDBConfig failed: %v", err)
		}
		if cfg.Type != DBTypeSQLite {
			t.Errorf("Expected default Type to be SQLite, got %v", cfg.Type)
		}
		expectedDefaultDSN := filepath.Join(tempDir, "glambdar.db")
		if cfg.DSN != expectedDefaultDSN {
			t.Errorf("Expected default DSN to be %s, got %s", expectedDefaultDSN, cfg.DSN)
		}
	})

	t.Run("SaveAndLoadCustomConfig", func(t *testing.T) {
		customCfg := &DBConfig{
			Type: DBTypePostgres,
			DSN:  "host=localhost user=test",
		}
		if err := SaveDBConfig(customCfg); err != nil {
			t.Fatalf("SaveDBConfig failed: %v", err)
		}

		cfg2, err := LoadDBConfig()
		if err != nil {
			t.Fatalf("LoadDBConfig failed: %v", err)
		}
		if cfg2.Type != DBTypePostgres {
			t.Errorf("Expected Type to be Postgres, got %v", cfg2.Type)
		}
		if cfg2.DSN != "host=localhost user=test" {
			t.Errorf("Expected DSN to be 'host=localhost user=test', got %s", cfg2.DSN)
		}
	})
}
