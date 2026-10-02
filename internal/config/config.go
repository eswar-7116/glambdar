package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"github.com/eswar-7116/glambdar/v3/internal/storage"
	"github.com/google/uuid"
)

type DBType int

const (
	DBTypePostgres DBType = iota
	DBTypeMySQL
)

func (t DBType) MarshalJSON() ([]byte, error) {
	switch t {
	case DBTypeMySQL:
		return json.Marshal("mysql")
	default:
		return json.Marshal("postgres")
	}
}

func (t *DBType) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	switch s {
	case "mysql":
		*t = DBTypeMySQL
	case "postgres":
		*t = DBTypePostgres
	default:
		return fmt.Errorf("unsupported db_type %q: must be \"postgres\" or \"mysql\"", s)
	}
	return nil
}

type Config struct {
	NodeID    string           `json:"node_id"`
	HTTPPort  string           `json:"http_port"`  // controller HTTP port (default: 8000)
	GRPCPort  string           `json:"grpc_port"`  // agent gRPC port
	RedisAddr string           `json:"redis_addr"` // Redis address for cluster coordination
	Type      DBType           `json:"db_type"`    // postgres, mysql
	DSN       string           `json:"dsn"`        // Data Source Name
	S3        storage.S3Config `json:"s3"`         // S3 storage config
}

type CLIOverrides struct {
	DBType            string
	DSN               string
	RedisAddr         string
	NodeID            string
	HTTPPort          string
	GRPCPort          string
	S3Endpoint        string
	S3Region          string
	S3Bucket          string
	S3AccessKeyID     string
	S3SecretAccessKey string
	S3SessionToken    string
	S3ForcePathStyle  *bool
}

var Overrides CLIOverrides

func LoadConfig() (*Config, error) {
	configPath := filepath.Join(ConfigDir, "config.json")
	file, err := os.Open(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf(
				"config file not found at %s: create it with a \"db_type\" of \"postgres\" or \"mysql\" and a valid \"dsn\"",
				configPath,
			)
		}
		return nil, err
	}
	defer file.Close()

	var config Config
	if err := json.NewDecoder(file).Decode(&config); err != nil {
		return nil, err
	}
	if config.NodeID == "" {
		config.NodeID, err = newNodeID()
		if err != nil {
			return nil, fmt.Errorf("failed to generate node ID: %w", err)
		}
		if err := SaveConfig(&config); err != nil {
			return nil, fmt.Errorf("failed to persist node ID: %w", err)
		}
	}

	// Override with environment variables if set
	if v := os.Getenv("GLMBD_DB_TYPE"); v != "" {
		switch v {
		case "postgres":
			config.Type = DBTypePostgres
		case "mysql":
			config.Type = DBTypeMySQL
		default:
			return nil, fmt.Errorf("unsupported GLMBD_DB_TYPE %q: must be \"postgres\" or \"mysql\"", v)
		}
	}
	if v := os.Getenv("GLMBD_DSN"); v != "" {
		config.DSN = v
	}
	if v := os.Getenv("GLMBD_REDIS_ADDR"); v != "" {
		config.RedisAddr = v
	}
	if v := os.Getenv("GLMBD_HTTP_PORT"); v != "" {
		config.HTTPPort = v
	}
	if v := os.Getenv("GLMBD_GRPC_PORT"); v != "" {
		config.GRPCPort = v
	}

	// S3 overrides
	if v := os.Getenv("GLMBD_S3_ENDPOINT"); v != "" {
		config.S3.Endpoint = v
	}
	if v := os.Getenv("GLMBD_S3_REGION"); v != "" {
		config.S3.Region = v
	}
	if v := os.Getenv("GLMBD_S3_BUCKET"); v != "" {
		config.S3.Bucket = v
	}
	if v := os.Getenv("GLMBD_S3_ACCESS_KEY_ID"); v != "" {
		config.S3.AccessKeyID = v
	}
	if v := os.Getenv("GLMBD_S3_SECRET_ACCESS_KEY"); v != "" {
		config.S3.SecretAccessKey = v
	}
	if v := os.Getenv("GLMBD_S3_SESSION_TOKEN"); v != "" {
		config.S3.SessionToken = v
	}
	if v := os.Getenv("GLMBD_S3_FORCE_PATH_STYLE"); v != "" {
		if b, err := strconv.ParseBool(v); err == nil {
			config.S3.ForcePathStyle = b
		}
	}

	// Override with CLI flags (highest precedence)
	// Read from overrides
	if Overrides.DBType != "" {
		switch Overrides.DBType {
		case "postgres":
			config.Type = DBTypePostgres
		case "mysql":
			config.Type = DBTypeMySQL
		default:
			return nil, fmt.Errorf("unsupported --db-type %q: must be \"postgres\" or \"mysql\"", Overrides.DBType)
		}
	}
	if Overrides.DSN != "" {
		config.DSN = Overrides.DSN
	}
	if Overrides.RedisAddr != "" {
		config.RedisAddr = Overrides.RedisAddr
	}
	if Overrides.NodeID != "" {
		config.NodeID = Overrides.NodeID
	}
	if Overrides.HTTPPort != "" {
		config.HTTPPort = Overrides.HTTPPort
	}
	if Overrides.GRPCPort != "" {
		config.GRPCPort = Overrides.GRPCPort
	}
	if Overrides.S3Endpoint != "" {
		config.S3.Endpoint = Overrides.S3Endpoint
	}
	if Overrides.S3Region != "" {
		config.S3.Region = Overrides.S3Region
	}
	if Overrides.S3Bucket != "" {
		config.S3.Bucket = Overrides.S3Bucket
	}
	if Overrides.S3AccessKeyID != "" {
		config.S3.AccessKeyID = Overrides.S3AccessKeyID
	}
	if Overrides.S3SecretAccessKey != "" {
		config.S3.SecretAccessKey = Overrides.S3SecretAccessKey
	}
	if Overrides.S3SessionToken != "" {
		config.S3.SessionToken = Overrides.S3SessionToken
	}
	if Overrides.S3ForcePathStyle != nil {
		config.S3.ForcePathStyle = *Overrides.S3ForcePathStyle
	}
	return &config, nil
}

func SaveConfig(config *Config) error {
	configPath := filepath.Join(ConfigDir, "config.json")
	file, err := os.Create(configPath)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	return encoder.Encode(config)
}

func newNodeID() (string, error) {
	id, err := uuid.NewRandom()
	if err != nil {
		return "", err
	}

	return id.String(), nil
}
