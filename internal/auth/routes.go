package auth

import "github.com/gin-gonic/gin"

// WARNING: Every new route added to Gin MUST be mapped here, or it will 403 silently.
var routeActions = map[string]string{
	// Function operations
	"/deploy":        "deploy",
	"/invoke/:name":  "invoke",
	"/del/:name":     "delete",
	"/config/:name":  "config",
	"/info":          "info",
	"/info/:name":    "info",
	"/logs/:name":    "logs",

	// Auth management
	"/auth/keys":     "auth",
	"/auth/keys/:id": "auth",
	"/auth/audit":    "auth",
}

// ResolveAction maps a Gin route path to an action
func ResolveAction(fullPath string) (string, bool) {
	action, ok := routeActions[fullPath]
	return action, ok
}

// RegisterRoutes registers auth API routes
func RegisterRoutes(router *gin.Engine) {
	authGroup := router.Group("/auth")
	{
		authGroup.POST("/keys", createKeyHandler)
		authGroup.GET("/keys", listKeysHandler)
		authGroup.PATCH("/keys/:id", updateKeyRoleHandler)
		authGroup.DELETE("/keys/:id", deleteKeyHandler)
		authGroup.GET("/audit", getAuditLogsHandler)
	}
}
