package functions

import (
	"time"

	"github.com/eswar-7116/glambdar/v3/internal/config"
)

type Metadata struct {
	Name          string    `json:"name" gorm:"primaryKey"`
	CreatedAt     time.Time `json:"createdAt"`
	LastInvokedAt time.Time `json:"lastInvokedAt"`
	InvokeCount   int       `json:"invokeCount"`
	RateLimit     int       `json:"rateLimit" gorm:"default:0"` // 0 = unlimited
	MaxConcurrency int32     `json:"maxConcurrency" gorm:"default:10"`
}

func LoadMetadata(funcName string) (*Metadata, error) {
	var m Metadata
	if err := config.DB.First(&m, "name = ?", funcName).Error; err != nil {
		return nil, err
	}
	return &m, nil
}

func SaveMetadata(m *Metadata) error {
	return config.DB.Save(m).Error
}

func DeleteMetadata(funcName string) error {
	return config.DB.Delete(&Metadata{}, "name = ?", funcName).Error
}

func GetAllMetadata() ([]Metadata, error) {
	var m []Metadata
	if err := config.DB.Find(&m).Error; err != nil {
		return nil, err
	}
	return m, nil
}
