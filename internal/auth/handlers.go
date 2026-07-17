package auth

import (
	"net/http"
	"strconv"
	"time"

	"github.com/eswar-7116/glambdar/v3/internal/config"
	"github.com/gin-gonic/gin"
)

type createKeyRequest struct {
	Name string `json:"name" binding:"required"`
	Role string `json:"role" binding:"required"`
}

type createKeyResponse struct {
	Name   string `json:"name"`
	Role   Role   `json:"role"`
	APIKey string `json:"apiKey"`
}

type updateRoleRequest struct {
	Role string `json:"role" binding:"required"`
}

func createKeyHandler(c *gin.Context) {
	var req createKeyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: name and role are required"})
		return
	}

	role, err := ValidateRole(req.Role)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var existing APIKey
	if err := config.DB.Where("name = ?", req.Name).First(&existing).Error; err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "a key with this name already exists"})
		return
	}

	rawKey, err := GenerateAPIKey(role)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate API key"})
		return
	}

	apiKey := APIKey{
		KeyHash:   HashKey(rawKey),
		KeyPrefix: KeyPrefixFromRaw(rawKey),
		Name:      req.Name,
		Role:      role,
		IsRoot:    false,
		CreatedAt: time.Now().UTC(),
	}
	if err := config.DB.Create(&apiKey).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save API key"})
		return
	}

	c.JSON(http.StatusCreated, createKeyResponse{
		Name:   req.Name,
		Role:   role,
		APIKey: rawKey,
	})
}

func listKeysHandler(c *gin.Context) {
	var keys []APIKey
	if err := config.DB.Find(&keys).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch keys"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"keys": keys})
}

func updateKeyRoleHandler(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid key ID"})
		return
	}

	var apiKey APIKey
	if err := config.DB.First(&apiKey, uint(id)).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "key not found"})
		return
	}

	if apiKey.IsRoot {
		c.JSON(http.StatusBadRequest, gin.H{"error": "cannot modify root key"})
		return
	}

	var req updateRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: role is required"})
		return
	}
	newRole, err := ValidateRole(req.Role)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	apiKey.Role = newRole
	if err := config.DB.Save(&apiKey).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update role"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":     apiKey.ID,
		"name":   apiKey.Name,
		"role":   apiKey.Role,
		"status": "updated",
	})
}

func deleteKeyHandler(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid key ID"})
		return
	}

	var apiKey APIKey
	if err := config.DB.First(&apiKey, uint(id)).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "key not found"})
		return
	}

	if apiKey.IsRoot {
		c.JSON(http.StatusBadRequest, gin.H{"error": "cannot delete root key"})
		return
	}

	if err := config.DB.Delete(&apiKey).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete key"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":     uint(id),
		"name":   apiKey.Name,
		"status": "revoked",
	})
}

func getAuditLogsHandler(c *gin.Context) {
	var logs []AuditLog
	if err := config.DB.Order("timestamp DESC").Limit(100).Find(&logs).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch audit logs"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"logs": logs})
}
