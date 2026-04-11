package functions

import (
	"time"

	"github.com/eswar-7116/glambdar/internal/config"
)

type Log struct {
	ID        uint      `json:"id" gorm:"primaryKey;autoIncrement"`
	FuncName  string    `json:"funcName" gorm:"not null"`
	InvokedAt time.Time `json:"invokedAt" gorm:"not null"`
	Stdout    string    `json:"stdout"`
	Stderr    string    `json:"stderr"`
}

func SaveLog(l *Log) error {
	return config.DB.Create(l).Error
}

func GetLogsByFunction(funcName string) ([]Log, error) {
	var logs []Log
	if err := config.DB.Where("func_name = ?", funcName).Order("invoked_at DESC").Find(&logs).Error; err != nil {
		return nil, err
	}
	return logs, nil
}

func DeleteLogsByFunction(funcName string) error {
	return config.DB.Where("func_name = ?", funcName).Delete(&Log{}).Error
}
