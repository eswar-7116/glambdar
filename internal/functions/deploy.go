package functions

import (
	"errors"
	"fmt"
	"time"

	"github.com/eswar-7116/glambdar/v3/internal/util"
	"gorm.io/gorm"
)

func Deploy(zipFilePath string, funcName string, rateLimit int) error {
	// Check if a function with this name already exists
	_, err := LoadMetadata(funcName)
	if err == nil {
		return fmt.Errorf("function '%s' already exists", funcName)
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return fmt.Errorf("error checking function existence: %w", err)
	}

	// Extract the zip
	_, err = util.ExtractZIP(zipFilePath, funcName)
	if err != nil {
		return err
	}

	// Initialize function metadata
	meta := Metadata{
		Name:           funcName,
		CreatedAt:      time.Now().UTC(),
		InvokeCount:    0,
		RateLimit:      rateLimit,
		MaxConcurrency: 10,
	}
	if err := SaveMetadata(&meta); err != nil {
		return err
	}

	return nil
}
