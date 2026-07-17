package functions

import (
	"github.com/eswar-7116/glambdar/v3/internal/config"
)

func UpdateRateLimit(funcName string, limit int) error {
	md, err := LoadMetadata(funcName)
	if err != nil {
		return err
	}

	md.RateLimit = limit
	if err := SaveMetadata(md); err != nil {
		return err
	}

	config.PoolManager.UpdateLimiter(funcName, limit)
	return nil
}
