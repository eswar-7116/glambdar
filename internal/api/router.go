package api

import (
	"net/http"

	"github.com/eswar-7116/glambdar/internal/auth"
	"github.com/gin-gonic/gin"
)

func Router() *gin.Engine {
	router := gin.Default()
	router.Use(gin.Recovery())

	// Unauthenticated healthcheck endpoint
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "healthy"})
	})

	router.Use(auth.AuthMiddleware())

	registerDeployRoutes(router)
	registerInvokeRoutes(router)
	registerInfoRoutes(router)
	registerDeleteRoutes(router)
	registerLogRoutes(router)
	registerConfigRoutes(router)
	auth.RegisterRoutes(router)

	return router
}
