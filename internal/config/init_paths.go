package config

import (
	"os"
	"path/filepath"

	"github.com/eswar-7116/glambdar/v3/internal/docker"
	"github.com/eswar-7116/glambdar/v3/internal/pool"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

var (
	ConfigDir    string
	FunctionsDir string
	WorkerPath   string
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
	WorkerPath = filepath.Join(ConfigDir, "worker", "glambdar-worker.js")

	DockerClient.WorkerPath = WorkerPath

	err := os.MkdirAll(FunctionsDir, 0755)
	if err != nil {
		return err
	}

	config, err := LoadConfig()
	if err != nil {
		return err
	}

	var dialector gorm.Dialector
	switch config.Type {
	case DBTypePostgres:
		dialector = postgres.Open(config.DSN)
	case DBTypeMySQL:
		dialector = mysql.Open(config.DSN)
	case DBTypeSQLite:
		fallthrough
	default:
		dialector = sqlite.Open(config.DSN)
	}

	DB, err = gorm.Open(dialector, &gorm.Config{})
	if err != nil {
		return err
	}

	if config.Type == DBTypeSQLite {
		DB.Exec("PRAGMA journal_mode=WAL;")
	}

	return nil
}
