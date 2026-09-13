package config

import (
	"encoding/json"
	"os"
	"testing"
)

func TestDBTypeJSON(t *testing.T) {
	tests := []struct {
		name     string
		dbType   DBType
		expected string
	}{
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

func TestConfigLoadSave(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "glambdar-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	originalConfigDir := ConfigDir
	ConfigDir = tempDir
	defer func() { ConfigDir = originalConfigDir }()

	t.Run("MissingConfigFile", func(t *testing.T) {
		_, err := LoadConfig()
		if err == nil {
			t.Fatal("expected LoadConfig to fail when config.json is missing")
		}
	})

	t.Run("SaveAndLoadCustomConfig", func(t *testing.T) {
		customCfg := &Config{
			Type:   DBTypePostgres,
			DSN:    "host=localhost user=test dbname=glambdar",
			NodeID: "550e8400-e29b-41d4-a716-446655440000",
		}
		if err := SaveConfig(customCfg); err != nil {
			t.Fatalf("SaveConfig failed: %v", err)
		}

		cfg2, err := LoadConfig()
		if err != nil {
			t.Fatalf("LoadConfig failed: %v", err)
		}
		if cfg2.Type != DBTypePostgres {
			t.Errorf("Expected Type to be Postgres, got %v", cfg2.Type)
		}
		if cfg2.DSN != "host=localhost user=test dbname=glambdar" {
			t.Errorf("Expected DSN to be 'host=localhost user=test dbname=glambdar', got %s", cfg2.DSN)
		}
		if cfg2.NodeID != customCfg.NodeID {
			t.Errorf("Expected NodeID to be %q, got %q", customCfg.NodeID, cfg2.NodeID)
		}
	})
}

func TestLoadConfigGeneratesAndPersistsNodeID(t *testing.T) {
	tempDir := t.TempDir()
	originalConfigDir := ConfigDir
	ConfigDir = tempDir
	defer func() { ConfigDir = originalConfigDir }()

	if err := SaveConfig(&Config{Type: DBTypePostgres, DSN: "test"}); err != nil {
		t.Fatalf("SaveConfig failed: %v", err)
	}

	first, err := LoadConfig()
	if err != nil {
		t.Fatalf("first LoadConfig failed: %v", err)
	}
	if first.NodeID == "" {
		t.Fatal("expected LoadConfig to generate a NodeID")
	}
	if len(first.NodeID) != 36 || first.NodeID[8] != '-' || first.NodeID[13] != '-' || first.NodeID[18] != '-' || first.NodeID[23] != '-' {
		t.Fatalf("expected NodeID to be UUID-shaped, got %q", first.NodeID)
	}
	if first.NodeID[14] != '4' {
		t.Errorf("expected UUID version 4, got %q", first.NodeID)
	}

	second, err := LoadConfig()
	if err != nil {
		t.Fatalf("second LoadConfig failed: %v", err)
	}
	if second.NodeID != first.NodeID {
		t.Errorf("expected NodeID to persist as %q, got %q", first.NodeID, second.NodeID)
	}
}
