package api

import "github.com/gin-gonic/gin"

func Router() *gin.Engine {
	router := gin.Default()
	router.Use(gin.Recovery())

	registerDeployRoutes(router)
	registerInvokeRoutes(router)
	registerInfoRoutes(router)
	registerDeleteRoutes(router)

	return router
}
