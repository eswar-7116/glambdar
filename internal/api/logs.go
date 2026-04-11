package api

import (
	"log"
	"net/http"

	"github.com/eswar-7116/glambdar/internal/functions"
	"github.com/gin-gonic/gin"
)

func registerLogRoutes(router *gin.Engine) {
	router.GET("/logs/:name", getLogsHandler)
}

func getLogsHandler(c *gin.Context) {
	name := c.Param("name")

	logs, err := functions.GetLogsByFunction(name)
	if err != nil {
		log.Printf("ERROR fetching logs for function '%s': %s", name, err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to fetch logs",
		})
		return
	}

	if len(logs) == 0 {
		// Check if function exists to decide between 404 and 200 []
		_, err := functions.LoadMetadata(name)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "Function not found!",
			})
			return
		}
	}

	c.JSON(http.StatusOK, logs)
}
