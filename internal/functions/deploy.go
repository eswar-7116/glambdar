package functions

import (
	"context"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/eswar-7116/glambdar/v3/internal/config"
	"gorm.io/gorm"
)

func Deploy(ctx context.Context, funcName string, zipReader io.Reader, rateLimit int) error {
	// Check if a function with this name already exists
	_, err := LoadMetadata(funcName)
	if err == nil {
		return fmt.Errorf("function '%s' already exists", funcName)
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return fmt.Errorf("error checking function existence: %w", err)
	}

	// Upload zip to S3 storage
	objectKey := funcName + ".zip"
	if config.StorageClient != nil {
		if err := config.StorageClient.Upload(ctx, objectKey, zipReader); err != nil {
			return fmt.Errorf("failed to upload zip to storage: %w", err)
		}
	} else {
		return fmt.Errorf("storage client is not initialized")
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
