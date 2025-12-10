package api

import "github.com/gin-gonic/gin"

func Router() *gin.Engine {
	router := gin.Default()

	router.GET("/", func(c *gin.Context) {
		c.Writer.Write([]byte("Hello from Glambdar!"))
	})

	return router
}
