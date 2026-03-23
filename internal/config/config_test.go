package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestInitPathsWithBase(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "glambdar-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

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

	if UDSPath != "/tmp/glambdar.sock" {
		t.Errorf("expected UDSPath to be /tmp/glambdar.sock, got %s", UDSPath)
	}

	expectedWorkerPath := filepath.Join(tempDir, "worker", "glambdar-worker.js")
	if WorkerPath != expectedWorkerPath {
		t.Errorf("expected WorkerPath to be %s, got %s", expectedWorkerPath, WorkerPath)
	}

	if DockerClient.UDSPath != UDSPath {
		t.Errorf("expected DockerClient.UDSPath to be %s, got %s", UDSPath, DockerClient.UDSPath)
	}

	if DockerClient.WorkerPath != expectedWorkerPath {
		t.Errorf("expected DockerClient.WorkerPath to be %s, got %s", expectedWorkerPath, DockerClient.WorkerPath)
	}
}

func TestInitPaths(t *testing.T) {
	// This test calls InitPaths which uses the user's home directory.
	// We just want to make sure it doesn't return an error.
	// In a real environment, we might want to mock os.UserHomeDir if possible,
	// but for now, a simple check for no error is enough.
	err := InitPaths()
	if err != nil {
		t.Errorf("InitPaths failed: %v", err)
	}
}
