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
	NodeID = config.NodeID

	var dialector gorm.Dialector
	switch config.Type {
	case DBTypePostgres:
		dialector = postgres.Open(config.DSN)
	case DBTypeMySQL:
		dialector = mysql.Open(config.DSN)
	default:
		return fmt.Errorf("unsupported database type: only \"postgres\" and \"mysql\" are supported")
	}

	DB, err = gorm.Open(dialector, &gorm.Config{})
	if err != nil {
		return err
	}

	if config.S3.Bucket != "" || config.S3.Endpoint != "" {
		s3Store, err := storage.NewS3Storage(config.S3)
		if err != nil {
			return err
		}
		StorageClient = s3Store
	}

	return nil
}
