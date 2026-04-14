package api

import (
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/eswar-7116/glambdar/internal/config"
	"github.com/eswar-7116/glambdar/internal/functions"
	"github.com/gin-gonic/gin"
)

func registerDeleteRoutes(router *gin.Engine) {
	router.DELETE("/del/:name", deleteFuncHandler)
}

func deleteFuncHandler(c *gin.Context) {
	name := c.Param("name")
	funcDir := filepath.Join(config.FunctionsDir, name)
	info, err := os.Stat(funcDir)
	if err != nil || !info.IsDir() {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Function not found!",
		})
		return
	}

	err = os.RemoveAll(funcDir)
	if err != nil {
		log.Printf("ERROR deleteing function files of '%s': %s\n", funcDir, err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to remove function files",
		})
		return
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
