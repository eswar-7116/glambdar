package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"flag"

	"github.com/eswar-7116/glambdar/v3/internal/storage"
)

type DBType int

const (
	DBTypeSQLite DBType = iota
	DBTypePostgres
	DBTypeMySQL
)

func (t DBType) MarshalJSON() ([]byte, error) {
	switch t {
	case DBTypePostgres:
		return json.Marshal("postgres")
	case DBTypeMySQL:
		return json.Marshal("mysql")
	case DBTypeSQLite:
		fallthrough
	default:
		return json.Marshal("sqlite")
	}
}

func (t *DBType) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	switch s {
	case "postgres":
		*t = DBTypePostgres
	case "mysql":
		*t = DBTypeMySQL
	case "sqlite":
		fallthrough
	default:
		*t = DBTypeSQLite
	}
	return nil
}

type Config struct {
	Type DBType           `json:"db_type"` // sqlite, postgres, mysql
	DSN  string           `json:"dsn"`     // Data Source Name
	S3   storage.S3Config `json:"s3"`      // S3 storage config
}

func LoadConfig() (*Config, error) {
	configPath := filepath.Join(ConfigDir, "config.json")

	file, err := os.Open(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			// Create default config
			defaultConfig := &Config{
				Type: DBTypeSQLite,
				DSN:  filepath.Join(ConfigDir, "glambdar.db"),
			}
			return defaultConfig, SaveConfig(defaultConfig)
		}
		return nil, err
	}
	defer file.Close()

	var config Config
	if err := json.NewDecoder(file).Decode(&config); err != nil {
		return nil, err
	}

	// Override with environment variables if set
	if v := os.Getenv("GLMBD_DB_TYPE"); v != "" {
		switch v {
		case "sqlite":
			config.Type = DBTypeSQLite
		case "postgres":
			config.Type = DBTypePostgres
		case "mysql":
			config.Type = DBTypeMySQL
		default:
			// leave as is if unrecognized
		}
	}
	if v := os.Getenv("GLMBD_DSN"); v != "" {
		config.DSN = v
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
	fs := flag.NewFlagSet("config", flag.ContinueOnError)
	var (
		flagDBType string
		flagDSN string
		flagS3Endpoint string
		flagS3Region string
		flagS3Bucket string
		flagS3AccessKeyID string
		flagS3SecretAccessKey string
		flagS3SessionToken string
		flagS3ForcePathStyle string
	)
	fs.StringVar(&flagDBType, "db_type", "", "Database type (sqlite, postgres, mysql)")
	fs.StringVar(&flagDSN, "dsn", "", "Database DSN")
	fs.StringVar(&flagS3Endpoint, "s3_endpoint", "", "S3 endpoint")
	fs.StringVar(&flagS3Region, "s3_region", "", "S3 region")
	fs.StringVar(&flagS3Bucket, "s3_bucket", "", "S3 bucket")
	fs.StringVar(&flagS3AccessKeyID, "s3_access_key_id", "", "S3 access key ID")
	fs.StringVar(&flagS3SecretAccessKey, "s3_secret_access_key", "", "S3 secret access key")
	fs.StringVar(&flagS3SessionToken, "s3_session_token", "", "S3 session token")
	fs.StringVar(&flagS3ForcePathStyle, "s3_force_path_style", "", "S3 force path style (true/false)")
	_ = fs.Parse(os.Args[1:])

	if flagDBType != "" {
		switch flagDBType {
		case "sqlite":
			config.Type = DBTypeSQLite
		case "postgres":
			config.Type = DBTypePostgres
		case "mysql":
			config.Type = DBTypeMySQL
		}
	}
	if flagDSN != "" {
		config.DSN = flagDSN
	}
	if flagS3Endpoint != "" {
		config.S3.Endpoint = flagS3Endpoint
	}
	if flagS3Region != "" {
		config.S3.Region = flagS3Region
	}
	if flagS3Bucket != "" {
		config.S3.Bucket = flagS3Bucket
	}
	if flagS3AccessKeyID != "" {
		config.S3.AccessKeyID = flagS3AccessKeyID
	}
	if flagS3SecretAccessKey != "" {
		config.S3.SecretAccessKey = flagS3SecretAccessKey
	}
	if flagS3SessionToken != "" {
		config.S3.SessionToken = flagS3SessionToken
	}
	if flagS3ForcePathStyle != "" {
		if b, err := strconv.ParseBool(flagS3ForcePathStyle); err == nil {
			config.S3.ForcePathStyle = b
		}
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
