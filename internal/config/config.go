package config

import (
	"os"
	"path/filepath"

	"github.com/eswar-7116/glambdar/internal/docker"
	"github.com/eswar-7116/glambdar/internal/pool"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

var (
	ConfigDir    string
	FunctionsDir string
	WorkerPath   string
	UDSPath      = "/tmp/glambdar.sock"
	DockerClient = &docker.Docker{}
	PoolManager  = &pool.PoolManager{}
	DB           *gorm.DB
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

	dbPath := filepath.Join(ConfigDir, "glambdar.db")
	DB, err = gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		return err
	}
	DB.Exec("PRAGMA journal_mode=WAL;")

	return nil
}
