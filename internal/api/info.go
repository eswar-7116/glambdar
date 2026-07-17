package api

import (
	"log"
	"net/http"

	"github.com/eswar-7116/glambdar/v3/internal/functions"
	"github.com/gin-gonic/gin"
)

func registerInfoRoutes(router *gin.Engine) {
	router.GET("/info", infoHandler)
	router.GET("/info/:name", functionInfoHandler)
}

func infoHandler(c *gin.Context) {
	deployedFunctions, err := functions.GetAllMetadata()
	if err != nil {
		log.Println("ERROR loading all metadata:", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to load function metadata",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"count":     len(deployedFunctions),
		"functions": deployedFunctions,
	})
}

func functionInfoHandler(c *gin.Context) {
	name := c.Param("name")

	md, err := functions.LoadMetadata(name)
	if err != nil {
		log.Printf("ERROR reading function '%s' details: %s", name, err)
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Function not found!",
		})
		return
	}

	c.JSON(http.StatusOK, md)
}
