package api

import (
	"errors"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/eswar-7116/glambdar/v3/internal/config"
	"github.com/eswar-7116/glambdar/v3/internal/functions"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func registerDeleteRoutes(router *gin.Engine) {
	router.DELETE("/del/:name", deleteFuncHandler)
}

func deleteFuncHandler(c *gin.Context) {
	name := c.Param("name")

	// Check if function exists in DB metadata
	_, err := functions.LoadMetadata(name)
	funcDir := filepath.Join(config.FunctionsDir, name)
	_, statErr := os.Stat(funcDir)

	if errors.Is(err, gorm.ErrRecordNotFound) && os.IsNotExist(statErr) {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Function not found!",
		})
		return
	}

	// Delete from S3 storage if configured
	if config.StorageClient != nil {
		objectKey := name + ".zip"
		if s3Err := config.StorageClient.Delete(c.Request.Context(), objectKey); s3Err != nil {
			log.Printf("ERROR deleting function zip from storage '%s': %s\n", objectKey, s3Err)
		}
	}

	// Remove local files if present
	if statErr == nil {
		if removeErr := os.RemoveAll(funcDir); removeErr != nil {
			log.Printf("ERROR deleting function files of '%s': %s\n", funcDir, removeErr)
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to remove function files",
			})
			return
		}
	}

	err = functions.DeleteMetadata(name)
	if err != nil {
		log.Printf("ERROR deleting function metadata of '%s': %s\n", name, err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to remove function metadata",
		})
		return
	}

	err = functions.DeleteLogsByFunction(name)
	if err != nil {
		log.Printf("ERROR deleting function logs of '%s': %s\n", name, err)
		// We don't return here because files and metadata are already gone
	}

	// Clean up container pool
	config.PoolManager.DeletePool(c, config.DockerClient, name)

	c.JSON(http.StatusOK, gin.H{
		"deleted": name,
	})
}
