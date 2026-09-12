package api

import (
	"bytes"
	"io"
	"log"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"

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

	c.JSON(http.StatusCreated, gin.H{
		"function": funcName,
		"status":   "deployed",
	})
}
