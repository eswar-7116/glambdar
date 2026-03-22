package util

import (
	"os"
	"path/filepath"
)

var (
	BaseDir      string
	FunctionsDir string
	WorkerPath   string
	UDSPath      = "/tmp/glambdar.sock"
)

func InitPaths() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	BaseDir = filepath.Join(home, ".glambdar")

	FunctionsDir = filepath.Join(BaseDir, "functions")

	_, err = os.Stat(FunctionsDir)
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
