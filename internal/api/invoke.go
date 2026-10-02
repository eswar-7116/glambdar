package api

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/eswar-7116/glambdar/v3/internal/cluster"
	"github.com/eswar-7116/glambdar/v3/internal/config"
	"github.com/eswar-7116/glambdar/v3/internal/functions"
	pb "github.com/eswar-7116/glambdar/v3/proto"
	"github.com/gin-gonic/gin"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func registerInvokeRoutes(router *gin.Engine) {
	for _, method := range []string{"GET", "POST", "PUT", "PATCH", "DELETE"} {
		router.Handle(method, "/invoke/:name", invokeHandler)
	}
}

func invokeHandler(c *gin.Context) {
	name := c.Param("name")

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

	if controllerMode {
		invokeViaGRPC(c, name, headers, bodyBytes)
		return
	}

	// Invoke locally
	req := functions.InvokeRequest{
		Method:  c.Request.Method,
		Headers: headers,
		Body:    string(bodyBytes),
	}

	resp, err := functions.Invoke(c, config.DockerClient, name, req)
	if err != nil {
		if errors.Is(err, functions.ErrRateLimited) {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error": "Rate limit exceeded. Please try again later.",
			})
			return
		}
		if os.IsNotExist(err) || errors.Is(err, os.ErrNotExist) || strings.Contains(err.Error(), "no such file") || strings.Contains(err.Error(), "file does not exist") || strings.Contains(err.Error(), "NoSuchKey") {
			funcDir := filepath.Join(config.FunctionsDir, name)
			abs, _ := filepath.Abs(funcDir)
			c.JSON(http.StatusNotFound, gin.H{
				"error":   "Function not found!",
				"funcDir": abs,
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

// Routes the invocation to an agent node via gRPC
func invokeViaGRPC(c *gin.Context, name string, headers map[string]string, body []byte) {
	node, err := clusterRouter.RouteInvocation(c.Request.Context(), name)
	if err != nil {
		if errors.Is(err, cluster.ErrNoNodesAvailable) {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"error": "No agent nodes available",
			})
			return
		}
		log.Println("ERROR routing invocation:", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to route invocation",
		})
		return
	}

	client, err := grpcPool.GetClient(node.Address)
	if err != nil {
		log.Printf("ERROR connecting to agent %s: %v", node.Address, err)
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "Failed to connect to agent node",
		})
		return
	}

	grpcResp, err := client.Invoke(c.Request.Context(), &pb.InvokeRequest{
		FuncName: name,
		Method:   c.Request.Method,
		Headers:  headers,
		Body:     body,
	})
	if err != nil {
		st, ok := status.FromError(err)
		if ok {
			switch st.Code() {
			case codes.ResourceExhausted:
				c.JSON(http.StatusTooManyRequests, gin.H{
					"error": "Rate limit exceeded. Please try again later.",
				})
			case codes.NotFound:
				c.JSON(http.StatusNotFound, gin.H{
					"error": "Function not found!",
				})
			case codes.Unavailable:
				c.JSON(http.StatusServiceUnavailable, gin.H{
					"error": "Agent node unavailable",
				})
			default:
				log.Printf("ERROR gRPC invoke on agent %s: %v", node.Address, err)
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "Glambdar Server Error",
				})
			}
			return
		}
		log.Printf("ERROR gRPC invoke on agent %s: %v", node.Address, err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Glambdar Server Error",
		})
		return
	}

	// Set response headers from gRPC response
	for k, v := range grpcResp.GetHeaders() {
		c.Header(k, v)
	}

	var respBody any
	if err := json.Unmarshal(grpcResp.GetBody(), &respBody); err != nil {
		log.Println("ERROR:", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(int(grpcResp.GetStatusCode()), respBody)
}
