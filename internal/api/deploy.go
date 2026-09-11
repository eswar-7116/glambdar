package api

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/eswar-7116/glambdar/v3/internal/config"
	"github.com/eswar-7116/glambdar/v3/internal/functions"
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

	// Create temporary directory if not exists
	tmpDir := filepath.Join(os.TempDir(), "glambdar")
	if err = os.MkdirAll(tmpDir, 0755); err != nil && !os.IsExist(err) {
		log.Println("ERROR while creating temporary directory: " + err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create temporary directory"})
		return
	}

	var zipBaseName string
	var zipFilePath string
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
			zipFilePath = filepath.Join(tmpDir, "glambdar-file-"+zipBaseName)
			zipFile, createErr := os.Create(zipFilePath)
			if createErr != nil {
				log.Println("ERROR while creating temporary upload: " + createErr.Error())
				c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save uploaded file"})
				return
			}
			_, copyErr := io.Copy(zipFile, part)
			closeErr := zipFile.Close()
			if copyErr != nil || closeErr != nil {
				logErr := copyErr
				if logErr == nil {
					logErr = closeErr
				}
				log.Println("ERROR while saving form file: " + logErr.Error())
				os.Remove(zipFilePath)
				c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save uploaded file"})
				return
			}
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
	defer os.Remove(zipFilePath)

	// Check if function already exists
	funcName := requestedFuncName
	if funcName == "" {
		funcName = strings.TrimSuffix(zipBaseName, filepath.Ext(zipBaseName))
	}
	funcDir := filepath.Join(config.FunctionsDir, funcName)
	if _, err = os.Stat(funcDir); err == nil {
		existsError := fmt.Sprintf("function directory '%s' already exists", funcDir)
		log.Println("ERROR: " + existsError)
		c.JSON(http.StatusBadRequest, gin.H{"error": existsError})
		return
	} else if !os.IsNotExist(err) {
		log.Println("ERROR while checking if function exists: " + err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to check if function directory exists"})
		return
	}

	// Deploy the function
	if err := functions.Deploy(zipFilePath, funcName, rateLimit); err != nil {
		log.Println("ERROR while deploying the function: " + err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to deploy function: " + err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"function": funcName,
		"status":   "deployed",
	})
}
