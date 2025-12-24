package api

import (
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/eswar-7116/glambdar/internal/functions"
	"github.com/eswar-7116/glambdar/internal/util"
	"github.com/gin-gonic/gin"
)

func registerInfoRoutes(router *gin.Engine) {
	router.GET("/info", infoHandler)
}

func infoHandler(c *gin.Context) {
	entries, err := os.ReadDir(util.FunctionsDir)
	if err != nil {
		log.Println("ERROR reading functions directory:", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Unable to read functions directory!",
		})
		return
	}

	var deployedFunctions []functions.Metadata
	for _, entry := range entries {
		md, err := functions.LoadMetadata(filepath.Join(util.FunctionsDir, entry.Name()))
		if err != nil {
			log.Println("ERROR reading function metadata of", entry.Name()+":", err)
			continue
		}
		deployedFunctions = append(deployedFunctions, *md)
	}

	c.JSON(http.StatusOK, gin.H{
		"count":     len(deployedFunctions),
		"functions": deployedFunctions,
	})
}
