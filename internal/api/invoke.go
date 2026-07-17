package api

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/eswar-7116/glambdar/v3/internal/config"
	"github.com/eswar-7116/glambdar/v3/internal/functions"
	"github.com/gin-gonic/gin"
)

func registerInvokeRoutes(router *gin.Engine) {
	for _, method := range []string{"GET", "POST", "PUT", "PATCH", "DELETE"} {
		router.Handle(method, "/invoke/:name", invokeHandler)
	}
}

func invokeHandler(c *gin.Context) {
	name := c.Param("name")
	funcDir := filepath.Join(config.FunctionsDir, name)

	// Check if function exists
	info, err := os.Stat(funcDir)
	if err != nil || !info.IsDir() {
		abs, _ := filepath.Abs(funcDir)
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "Function not found!",
			"funcDir": abs,
		})
		return
	}

	// Read request headers
	headers := make(map[string]string)
	for k, v := range c.Request.Header {
		if k == "Content-Length" ||
			k == "Transfer-Encoding" ||
			k == "Connection" {
			continue
		}
		headers[k] = strings.Join(v, ",")
	}

	// Read request body
	var bodyBytes []byte
	if c.Request.Body != nil {
		var err error
		bodyBytes, err = c.GetRawData()
		if err != nil {
			log.Println("ERROR: " + err.Error())
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Failed to read request",
			})
			return
		}
	}

	// Build request object
	req := functions.InvokeRequest{
		Method:  c.Request.Method,
		Headers: headers,
		Body:    string(bodyBytes),
	}

	// Invoke function and get response object
	resp, err := functions.Invoke(c, config.DockerClient, name, req)
	if err != nil {
		if errors.Is(err, functions.ErrRateLimited) {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error": "Rate limit exceeded. Please try again later.",
			})
			return
		}
		log.Println("ERROR:", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Glambdar Server Error",
		})
		return
	}

	// Set response headers
	for k, v := range resp.Headers {
		c.Header(k, v)
	}

	var body any
	if err := json.Unmarshal(resp.Body, &body); err != nil {
		log.Println("ERROR:", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	// Set response status and body
	c.JSON(resp.StatusCode, body)
}
