package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestInitPathsWithBase(t *testing.T) {
	dsn := os.Getenv("TEST_DSN")
	if dsn == "" {
		t.Skip("TEST_DSN not set. Skipping InitPathsWithBase test")
	}

	tempDir, err := os.MkdirTemp("", "glambdar-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	ConfigDir = tempDir
	if err := SaveConfig(&Config{Type: DBTypePostgres, DSN: dsn}); err != nil {
		t.Fatalf("failed to write temp config: %v", err)
	}

	err = InitPathsWithBase(tempDir)
	if err != nil {
		t.Fatalf("InitPathsWithBase failed: %v", err)
	}

	if ConfigDir != tempDir {
		t.Errorf("expected BaseDir to be %s, got %s", tempDir, ConfigDir)
	}

	expectedFunctionsDir := filepath.Join(tempDir, "functions")
	if FunctionsDir != expectedFunctionsDir {
		t.Errorf("expected FunctionsDir to be %s, got %s", expectedFunctionsDir, FunctionsDir)
	}

	if _, err := os.Stat(expectedFunctionsDir); os.IsNotExist(err) {
		t.Errorf("expected FunctionsDir to be created, but it does not exist")
	}

	expectedWorkerPath := filepath.Join(tempDir, "worker", "glambdar-worker.js")
	if WorkerPath != expectedWorkerPath {
		t.Errorf("expected WorkerPath to be %s, got %s", expectedWorkerPath, WorkerPath)
	}

	if DockerClient.WorkerPath != expectedWorkerPath {
		t.Errorf("expected DockerClient.WorkerPath to be %s, got %s", expectedWorkerPath, DockerClient.WorkerPath)
	}
}

func TestInitPaths(t *testing.T) {
	dsn := os.Getenv("TEST_DSN")
	if dsn == "" {
		t.Skip("TEST_DSN not set. Skipping InitPaths test")
	}

	tempHome, _ := os.MkdirTemp("", "home-*")
	defer os.RemoveAll(tempHome)
	t.Setenv("HOME", tempHome)

	configDir := filepath.Join(tempHome, ".glambdar")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("failed to create config dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(configDir, "config.json"), []byte(`{"db_type":"postgres","dsn":"`+dsn+`"}`), 0600); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	err := InitPaths()
	if err != nil {
		t.Errorf("InitPaths failed: %v", err)
	}
}
