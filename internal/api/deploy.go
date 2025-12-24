package api

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/eswar-7116/glambdar/internal/functions"
	"github.com/eswar-7116/glambdar/internal/util"
	"github.com/gin-gonic/gin"
)

func registerDeployRoutes(router *gin.Engine) {
	router.POST("/deploy", deployHandler)
}

func deployHandler(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		log.Println("ERROR while receiving form file: " + err.Error())
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing zip file"})
		return
	}

	// Create temporary directory if not exists
	tmpDir := filepath.Join(os.TempDir(), "glambdar")
	if err = os.Mkdir(tmpDir, 0755); err != nil && !os.IsExist(err) {
		log.Println("ERROR while creating temporary directory: " + err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create temporary directory"})
		return
	}

	// Save zip file in the temporary directory
	zipBaseName := file.Filename
	zipFilePath := filepath.Join(tmpDir, "glambdar-file-"+zipBaseName)
	if err = c.SaveUploadedFile(file, zipFilePath); err != nil {
		log.Println("ERROR while saving form file: " + err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save uploaded file"})
		return
	}

	// Check if function already exists
	funcName := strings.TrimSuffix(zipBaseName, filepath.Ext(zipBaseName))
	funcDir := filepath.Join(util.FunctionsDir, funcName)
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
	if err := functions.Deploy(zipFilePath, funcName); err != nil {
		log.Println("ERROR while deploying the function: " + err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to check if function directory exists"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"function": funcName,
		"status":   "deployed",
	})
}
