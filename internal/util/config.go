package util

import (
	"fmt"
	"os"
	"path/filepath"
)

var (
	FunctionsDir string
	WorkerPath   string
	UDSPath      = "/tmp/glambdar.sock"
)

func InitPaths() error {
	BaseDir := os.Getenv("GLAMBDAR_DIR")
	if BaseDir == "" {
		return fmt.Errorf("GLAMBDAR_DIR is required")
	}

	FunctionsDir = filepath.Join(BaseDir, "functions")
	WorkerPath = filepath.Join(BaseDir, "worker", "glambdar-worker.js")
	return nil
}
