package api

import (
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/eswar-7116/glambdar/internal/util"
	"github.com/gin-gonic/gin"
)

func registerDeleteRoutes(router *gin.Engine) {
	router.DELETE("/del/:name", deleteFuncHandler)
}

func deleteFuncHandler(c *gin.Context) {
	name := c.Param("name")
	funcDir := filepath.Join(util.FunctionsDir, name)
	info, err := os.Stat(funcDir)
	if err != nil || !info.IsDir() {
		c.JSON(http.StatusNoContent, gin.H{
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

	c.JSON(http.StatusOK, gin.H{
		"deleted": name,
	})
}
