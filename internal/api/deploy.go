package api

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

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

	tmpDir := filepath.Join(os.TempDir(), "glambdar")
	if err = os.Mkdir(tmpDir, 0755); err != nil && !os.IsExist(err) {
		log.Println("ERROR while creating temporary directory: " + err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create temporary directory"})
		return
	}

	zipBaseName := file.Filename
	zipFilePath := filepath.Join(tmpDir, "glambdar-file-"+zipBaseName)
	if err = c.SaveUploadedFile(file, zipFilePath); err != nil {
		log.Println("ERROR while saving form file: " + err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save uploaded file"})
		return
	}

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

	funcPath, err := util.ExtractZIP(zipFilePath, funcName)
	if err != nil {
		log.Println("ERROR while extracting zip: " + err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to extract zip file"})
		return
	}

	meta := functions.Metadata{
		Name:        funcName,
		CreatedAt:   time.Now().UTC(),
		InvokeCount: 0,
	}
	functions.SaveMetadata(funcPath, &meta)

	c.JSON(http.StatusOK, gin.H{
		"function": funcName,
		"path":     funcPath,
		"status":   "deployed",
	})
}
