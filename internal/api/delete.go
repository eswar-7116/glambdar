package api

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/eswar-7116/glambdar/v3/internal/config"
	"github.com/eswar-7116/glambdar/v3/internal/functions"
	pb "github.com/eswar-7116/glambdar/v3/proto"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func registerDeleteRoutes(router *gin.Engine) {
	router.DELETE("/del/:name", deleteFuncHandler)
}

func deleteFuncHandler(c *gin.Context) {
	name := c.Param("name")

	// Check if function exists in DB metadata
	_, err := functions.LoadMetadata(name)
	funcDir := filepath.Join(config.FunctionsDir, name)
	_, statErr := os.Stat(funcDir)

	if errors.Is(err, gorm.ErrRecordNotFound) && os.IsNotExist(statErr) {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Function not found!",
		})
		return
	}

	// Delete from S3 storage if configured
	if config.StorageClient != nil {
		objectKey := name + ".zip"
		if s3Err := config.StorageClient.Delete(c.Request.Context(), objectKey); s3Err != nil {
			log.Printf("ERROR deleting function zip from storage '%s': %s\n", objectKey, s3Err)
		}
	}

	// Remove local files if present
	if statErr == nil {
		if removeErr := os.RemoveAll(funcDir); removeErr != nil {
			log.Printf("ERROR deleting function files of '%s': %s\n", funcDir, removeErr)
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to remove function files",
			})
			return
		}
	}

	err = functions.DeleteMetadata(name)
	if err != nil {
		log.Printf("ERROR deleting function metadata of '%s': %s\n", name, err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to remove function metadata",
		})
		return
	}

	err = functions.DeleteLogsByFunction(name)
	if err != nil {
		log.Printf("ERROR deleting function logs of '%s': %s\n", name, err)
		// We don't return here because files and metadata are already gone
	}

	// Clean up container pool (in standalone mode)
	if !controllerMode {
		config.PoolManager.DeletePool(c, config.DockerClient, name)
	}

	// Fan out EvictFunction to all healthy agents in controller mode
	if controllerMode && grpcPool != nil && stateProvider != nil {
		go fanOutEvict(name)
	}

	c.JSON(http.StatusOK, gin.H{
		"deleted": name,
	})
}

// Sends EvictFunction RPCs to all healthy agent nodes
func fanOutEvict(funcName string) {
	ctx := context.Background()
	nodes, err := stateProvider.GetHealthyNodes(ctx)
	if err != nil {
		log.Printf("WARNING: failed to get healthy nodes for evict fan-out: %v", err)
		return
	}

	for _, node := range nodes {
		client, err := grpcPool.GetClient(node.Address)
		if err != nil {
			log.Printf("WARNING: failed to connect to agent %s for evict: %v", node.Address, err)
			continue
		}
		if _, err := client.EvictFunction(ctx, &pb.EvictRequest{FuncName: funcName}); err != nil {
			log.Printf("WARNING: failed to evict function %s on agent %s: %v", funcName, node.Address, err)
		}
	}
}
