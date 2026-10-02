package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/eswar-7116/glambdar/v3/internal/docker"
	"github.com/eswar-7116/glambdar/v3/internal/pool"
	"github.com/eswar-7116/glambdar/v3/internal/storage"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var (
	ConfigDir     string
	FunctionsDir  string
	WorkerPath    string
	NodeID        string
	DockerClient  = &docker.Docker{}
	PoolManager   = &pool.PoolManager{}
	DB            *gorm.DB
	StorageClient storage.Storage
)

func InitPaths() error {
	return withHomeDir(InitPathsWithBase)
}

func InitAgentPaths() error {
	return withHomeDir(InitAgentPathsWithBase)
}

func withHomeDir(initFn func(string) error) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	return initFn(filepath.Join(home, ".glambdar"))
}

func initCore(baseDir string) (*Config, error) {
	ConfigDir = baseDir
	FunctionsDir = filepath.Join(ConfigDir, "functions")
	WorkerPath = filepath.Join(ConfigDir, "worker", "glambdar-worker.js")
	DockerClient.WorkerPath = WorkerPath

	if err := os.MkdirAll(FunctionsDir, 0755); err != nil {
		return nil, err
	}

	cfg, err := LoadConfig()
	if err != nil {
		return nil, err
	}
	NodeID = cfg.NodeID

	if cfg.S3.Bucket != "" || cfg.S3.Endpoint != "" {
		s3Store, err := storage.NewS3Storage(cfg.S3)
		if err != nil {
			return nil, err
		}
		StorageClient = s3Store
	}

	return cfg, nil
}

func connectDB(cfg *Config) (*gorm.DB, error) {
	var dialector gorm.Dialector
	switch cfg.Type {
	case DBTypePostgres:
		dialector = postgres.Open(cfg.DSN)
	case DBTypeMySQL:
		dialector = mysql.Open(cfg.DSN)
	default:
		return nil, fmt.Errorf("unsupported database type: only \"postgres\" and \"mysql\" are supported")
	}

	return gorm.Open(dialector, &gorm.Config{})
}

func InitPathsWithBase(baseDir string) error {
	cfg, err := initCore(baseDir)
	if err != nil {
		return err
	}

	db, err := connectDB(cfg)
	if err != nil {
		return err
	}
	DB = db
	return nil
}

func InitAgentPathsWithBase(baseDir string) error {
	cfg, err := initCore(baseDir)
	if err != nil {
		// For agent, allow running with defaults if config file is not present
		ConfigDir = baseDir
		FunctionsDir = filepath.Join(ConfigDir, "functions")
		WorkerPath = filepath.Join(ConfigDir, "worker", "glambdar-worker.js")
		DockerClient.WorkerPath = WorkerPath
		_ = os.MkdirAll(FunctionsDir, 0755)
		return nil
	}

	// Connect to DB if DSN is configured, but don't fail if absent
	if cfg.DSN != "" {
		if db, err := connectDB(cfg); err == nil {
			DB = db
		}
	}
	return nil
}

func InitControllerPaths() error {
	return withHomeDir(InitControllerPathsWithBase)
}

func InitControllerPathsWithBase(baseDir string) error {
	cfg, err := initCore(baseDir)
	if err != nil {
		return err
	}

	// Controller must have a DB connection
	db, err := connectDB(cfg)
	if err != nil {
		return err
	}
	DB = db

	// Controller doesn't need DockerClient or PoolManager
	return nil
}
