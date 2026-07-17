package auth

import (
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/eswar-7116/glambdar/v3/internal/config"
	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

var ipLimiters sync.Map
var auditLogChan = make(chan AuditLog, 1000)

func init() {
	go func() {
		for logEntry := range auditLogChan {
			config.DB.Create(&logEntry)
		}
	}()
}

func getIPLimiter(ip string) *rate.Limiter {
	limiter, exists := ipLimiters.Load(ip)
	if !exists {
		newLimiter := rate.NewLimiter(rate.Limit(5), 2)
		limiter, _ = ipLimiters.LoadOrStore(ip, newLimiter)
	}
	return limiter.(*rate.Limiter)
}

// AuthMiddleware intercepts and authorizes requests
func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if strings.HasPrefix(c.FullPath(), "/auth") && gin.Mode() != gin.TestMode {
			limiter := getIPLimiter(c.ClientIP())
			if !limiter.Allow() {
				c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
					"error": "rate limit exceeded on auth endpoint",
				})
				return
			}
		}

		rawKey := c.GetHeader("X-API-Key")
		if rawKey == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing API key"})
			return
		}

		keyHash := HashKey(rawKey)
		var apiKey APIKey
		if err := config.DB.Where("key_hash = ?", keyHash).First(&apiKey).Error; err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid API key"})
			return
		}

		action, ok := ResolveAction(c.FullPath())
		if !ok {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "route not authorized"})
			return
		}

		if !HasPermission(apiKey.Role, action, c.Request.Method) {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error": fmt.Sprintf("insufficient permissions: %s cannot %s %s",
					apiKey.Role, c.Request.Method, c.FullPath()),
			})
			return
		}

		c.Set("auth_user", &apiKey)
		c.Next()

		select {
		case auditLogChan <- AuditLog{
			Timestamp: time.Now().UTC(),
			ActorName: apiKey.Name,
			ActorRole: string(apiKey.Role),
			Action:    action,
			Method:    c.Request.Method,
			Path:      c.Request.URL.Path,
			Status:    c.Writer.Status(),
		}:
		default: // Non-blocking if channel is full
		}
	}
}
