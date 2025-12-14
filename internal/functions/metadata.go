package functions

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

type Metadata struct {
	Name          string    `json:"name"`
	CreatedAt     time.Time `json:"createdAt"`
	LastInvokedAt time.Time `json:"lastInvokedAt"`
	InvokeCount   int       `json:"invokeCount"`
}

func metaPath(funcDir string) string {
	return filepath.Join(funcDir, "meta.json")
}

func LoadMetadata(funcDir string) (*Metadata, error) {
	path := metaPath(funcDir)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var m Metadata
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, err
	}
	return &m, nil
}

func SaveMetadata(funcDir string, m *Metadata) error {
	path := metaPath(funcDir)
	bytes, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, bytes, 0644)
}
