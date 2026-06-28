package config

import (
	"encoding/json"
	"os"
	"path/filepath"
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

type DBConfig struct {
	Type DBType `json:"type"` // sqlite, postgres, mysql
	DSN  string `json:"dsn"`  // Data Source Name
}

func LoadDBConfig() (*DBConfig, error) {
	configPath := filepath.Join(ConfigDir, "db_config.json")

	file, err := os.Open(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			// Create default config
			defaultConfig := &DBConfig{
				Type: DBTypeSQLite,
				DSN:  filepath.Join(ConfigDir, "glambdar.db"),
			}
			return defaultConfig, SaveDBConfig(defaultConfig)
		}
		return nil, err
	}
	defer file.Close()

	var dbConfig DBConfig
	if err := json.NewDecoder(file).Decode(&dbConfig); err != nil {
		return nil, err
	}

	return &dbConfig, nil
}

func SaveDBConfig(config *DBConfig) error {
	configPath := filepath.Join(ConfigDir, "db_config.json")
	file, err := os.Create(configPath)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	return encoder.Encode(config)
}
