package config

import (
	"os"
	"path/filepath"

	"github.com/eswar-7116/glambdar/internal/docker"
	"github.com/eswar-7116/glambdar/internal/pool"
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

	dbConfig, err := LoadDBConfig()
	if err != nil {
		return err
	}

	var dialector gorm.Dialector
	switch dbConfig.Type {
	case DBTypePostgres:
		dialector = postgres.Open(dbConfig.DSN)
	case DBTypeMySQL:
		dialector = mysql.Open(dbConfig.DSN)
	case DBTypeSQLite:
		fallthrough
	default:
		dialector = sqlite.Open(dbConfig.DSN)
	}

	DB, err = gorm.Open(dialector, &gorm.Config{})
	if err != nil {
		return err
	}

	if dbConfig.Type == DBTypeSQLite {
		DB.Exec("PRAGMA journal_mode=WAL;")
	}

	return nil
}
