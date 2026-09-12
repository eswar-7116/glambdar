package testutil

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/url"
	"os"
	"strings"
	"testing"

	"github.com/eswar-7116/glambdar/v3/internal/config"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func RequireTestDSN(t *testing.T) string {
	t.Helper()

	dsn := os.Getenv("TEST_DSN")
	if dsn == "" {
		t.Skip("TEST_DSN not set; skipping DB test (set TEST_DSN=postgres://... to run)")
	}
	return dsn
}

func ResetTestDB(t *testing.T) {
	t.Helper()
	if config.DB == nil {
		return
	}

	tables := []string{"api_keys", "audit_logs", "logs", "metadata"}
	for _, table := range tables {
		if !config.DB.Migrator().HasTable(table) {
			continue
		}
		if err := config.DB.Exec("DELETE FROM " + table).Error; err != nil {
			t.Fatalf("failed to delete rows from %s: %v", table, err)
		}
	}
}

// SetupTestConfig writes a temporary config file for testing
func SetupTestConfig(t *testing.T, baseDir string) {
	t.Helper()

	dsn := isolatedPostgresDSN(t, RequireTestDSN(t), baseDir)
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		t.Fatalf("failed to create test config dir: %v", err)
	}
	prevConfigDir := config.ConfigDir
	config.ConfigDir = baseDir
	t.Cleanup(func() { config.ConfigDir = prevConfigDir })

	if err := config.SaveConfig(&config.Config{Type: config.DBTypePostgres, DSN: dsn}); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}
	if err := config.InitPathsWithBase(baseDir); err != nil {
		t.Fatalf("failed to initialize test config: %v", err)
	}
	ResetTestDB(t)
}

// SetupTestDB opens a PostgreSQL connection for testing using the TEST_DSN environment variable (skipped if not set)
func SetupTestDB(t *testing.T) {
	t.Helper()

	dsn := isolatedPostgresDSN(t, RequireTestDSN(t), t.Name())

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("testutil.SetupTestDB: failed to connect to test database: %v\n\nMake sure TEST_DSN points to a running PostgreSQL instance.", err)
	}

	config.DB = db
	ResetTestDB(t)

	// Drop all tables
	t.Cleanup(func() {
		sqlDB, err := db.DB()
		if err == nil {
			sqlDB.Close()
		}
	})
}

// IsolatedPostgresDSN returns a DSN that uses a unique schema for a test
func isolatedPostgresDSN(t *testing.T, dsn, identity string) string {
	t.Helper()

	hash := sha256.Sum256([]byte(identity))
	schema := "test_" + hex.EncodeToString(hash[:])[:16]
	adminDB, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("failed to connect to test database: %v", err)
	}
	if err := adminDB.Exec("CREATE SCHEMA IF NOT EXISTS " + schema).Error; err != nil {
		t.Fatalf("failed to create test schema %s: %v", schema, err)
	}
	t.Cleanup(func() {
		adminDB.Exec("DROP SCHEMA IF EXISTS " + schema + " CASCADE")
		if sqlDB, err := adminDB.DB(); err == nil {
			sqlDB.Close()
		}
	})

	if strings.HasPrefix(dsn, "postgres://") || strings.HasPrefix(dsn, "postgresql://") {
		parsed, err := url.Parse(dsn)
		if err != nil {
			t.Fatalf("failed to parse test database URL: %v", err)
		}
		query := parsed.Query()
		query.Set("options", fmt.Sprintf("-c search_path=%s", schema))
		parsed.RawQuery = query.Encode()
		return parsed.String()
	}

	return dsn + " options='-c search_path=" + schema + "'"
}
