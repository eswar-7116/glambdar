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

	_, err := os.Stat(FunctionsDir)
	if os.IsNotExist(err) {
		err = os.Mkdir(FunctionsDir, 0755)
		if err != nil {
			return err
		}
	} else if err != nil {
		return err
	}

	WorkerPath = filepath.Join(BaseDir, "worker", "glambdar-worker.js")
	return nil
}
