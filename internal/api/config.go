package api

import (
	"net/http"

	"github.com/eswar-7116/glambdar/internal/functions"
	"github.com/gin-gonic/gin"
)

type configRequest struct {
	RateLimit int `json:"rateLimit"`
}

func registerConfigRoutes(router *gin.Engine) {
	router.POST("/config/:name", configHandler)
}

func configHandler(c *gin.Context) {
	name := c.Param("name")
	var req configRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	if err := functions.UpdateRateLimit(name, req.RateLimit); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"function":  name,
		"rateLimit": req.RateLimit,
		"status":    "updated",
	})
}
