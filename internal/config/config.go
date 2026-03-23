package config

import (
	"os"
	"path/filepath"

	"github.com/eswar-7116/glambdar/internal/docker"
)

var (
	ConfigDir    string
	FunctionsDir string
	WorkerPath   string
	UDSPath      = "/tmp/glambdar.sock"
	DockerClient = &docker.Docker{}
)

func InitPaths() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	return InitPathsWithBase(filepath.Join(home, ".glambdar"))
}

func InitPathsWithBase(baseDir string) error {
	ConfigDir = baseDir
	FunctionsDir = filepath.Join(ConfigDir, "functions")
	UDSPath = "/tmp/glambdar.sock"
	WorkerPath = filepath.Join(ConfigDir, "worker", "glambdar-worker.js")

	DockerClient.UDSPath = UDSPath
	DockerClient.WorkerPath = WorkerPath

	err := os.MkdirAll(FunctionsDir, 0755)
	if err != nil {
		return err
	}

	return nil
}
