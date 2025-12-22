package functions

import (
	"time"

	"github.com/eswar-7116/glambdar/internal/util"
)

func Deploy(zipFilePath string, funcName string) error {
	// Extract the zip
	funcPath, err := util.ExtractZIP(zipFilePath, funcName)
	if err != nil {
		return err
	}

	// Initialize function metadata
	meta := Metadata{
		Name:        funcName,
		CreatedAt:   time.Now().UTC(),
		InvokeCount: 0,
	}
	SaveMetadata(funcPath, &meta)

	return nil
}
