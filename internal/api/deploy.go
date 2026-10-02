package api

import (
	"bytes"
	"context"
	"io"
	"log"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/eswar-7116/glambdar/v3/internal/functions"
	pb "github.com/eswar-7116/glambdar/v3/proto"
	"github.com/gin-gonic/gin"
)

func registerDeployRoutes(router *gin.Engine) {
	router.POST("/deploy", deployHandler)
}

func deployHandler(c *gin.Context) {
	reader, err := c.Request.MultipartReader()
	if err != nil {
		log.Println("ERROR while receiving form file: " + err.Error())
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing zip file"})
		return
	}

	var zipBaseName string
	var zipReader io.Reader
	var requestedFuncName string
	rateLimit := 0
	fileFound := false

	for {
		part, partErr := reader.NextPart()
		if partErr == io.EOF {
			break
		}
		if partErr != nil {
			log.Println("ERROR while reading multipart form: " + partErr.Error())
			c.JSON(http.StatusBadRequest, gin.H{"error": "missing zip file"})
			return
		}

		switch part.FormName() {
		case "file":
			if fileFound {
				continue
			}
			zipBaseName = filepath.Base(part.FileName())
			if zipBaseName == "." || zipBaseName == "" {
				continue
			}
			// Buffer the part reader
			buf := new(bytes.Buffer)
			if _, copyErr := io.Copy(buf, part); copyErr != nil {
				log.Println("ERROR while streaming form file: " + copyErr.Error())
				c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read uploaded file"})
				return
			}
			zipReader = buf
			fileFound = true
		case "funcName":
			value, readErr := io.ReadAll(part)
			if readErr != nil {
				log.Println("ERROR while reading function name: " + readErr.Error())
				c.JSON(http.StatusBadRequest, gin.H{"error": "invalid function name"})
				return
			}
			requestedFuncName = string(value)
		case "rateLimit":
			value, readErr := io.ReadAll(part)
			if readErr == nil {
				if parsed, parseErr := strconv.Atoi(string(value)); parseErr == nil {
					rateLimit = parsed
				}
			}
		}
	}

	if !fileFound {
		log.Println("ERROR while receiving form file: missing file part")
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing zip file"})
		return
	}

	funcName := requestedFuncName
	if funcName == "" {
		funcName = strings.TrimSuffix(zipBaseName, filepath.Ext(zipBaseName))
	}

	// Deploy the function (uploads directly to S3 without using /tmp)
	if err := functions.Deploy(c.Request.Context(), funcName, zipReader, rateLimit); err != nil {
		log.Println("ERROR while deploying the function: " + err.Error())
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to deploy function: " + err.Error()})
		return
	}

	// Fan out PreloadFunction to all healthy agents in controller mode
	if controllerMode && grpcPool != nil && stateProvider != nil {
		go fanOutPreload(funcName)
	}

	c.JSON(http.StatusCreated, gin.H{
		"function": funcName,
		"status":   "deployed",
	})
}

// Sends PreloadFunction RPCs to all healthy agent nodes
func fanOutPreload(funcName string) {
	ctx := context.Background()
	nodes, err := stateProvider.GetHealthyNodes(ctx)
	if err != nil {
		log.Printf("WARNING: failed to get healthy nodes for preload fan-out: %v", err)
		return
	}

	for _, node := range nodes {
		client, err := grpcPool.GetClient(node.Address)
		if err != nil {
			log.Printf("WARNING: failed to connect to agent %s for preload: %v", node.Address, err)
			continue
		}
		if _, err := client.PreloadFunction(ctx, &pb.PreloadRequest{FuncName: funcName}); err != nil {
			log.Printf("WARNING: failed to preload function %s on agent %s: %v", funcName, node.Address, err)
		}
	}
}
