package functions

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"
	"time"

	"github.com/eswar-7116/glambdar/internal/config"
	"github.com/eswar-7116/glambdar/internal/docker"
	"github.com/eswar-7116/glambdar/internal/pool"
	"github.com/moby/moby/api/pkg/stdcopy"
)

var ErrRateLimited = errors.New("rate limit exceeded")

type InvokeRequest struct {
	Method  string            `json:"method"`
	Headers map[string]string `json:"headers"`
	Body    string            `json:"body"`
}

type InvokeResponse struct {
	StatusCode int               `json:"statusCode"`
	Headers    map[string]string `json:"headers"`
	Body       json.RawMessage   `json:"body"`
	ColdStart  bool              `json:"coldStart"`
}

func Invoke(ctx context.Context, d *docker.Docker, funcName string, req InvokeRequest) (InvokeResponse, error) {
	funcDir, err := filepath.Abs(filepath.Join(config.FunctionsDir, funcName))
	if err != nil {
		return InvokeResponse{}, err
	}

	// Check if the function's directory exists
	info, err := os.Stat(funcDir)
	if err != nil {
		return InvokeResponse{}, err
	}
	if !info.IsDir() {
		return InvokeResponse{}, fmt.Errorf("%s is not a directory", funcDir)
	}

	// Load metadata
	md, err := LoadMetadata(funcName)
	if err != nil {
		return InvokeResponse{}, err
	}

	// Acquire a warm container or create a new one
	p := config.PoolManager.GetOrCreate(funcName, md.RateLimit, md.MaxConcurrency)
	if !p.Limiter.Allow() {
		return InvokeResponse{}, ErrRateLimited
	}

	e, warm := p.Acquire()
	if !warm {
		// Generate a per-container socket directory on the host
		socketDir, err := os.MkdirTemp("", "glambdar-sock-*")
		if err != nil {
			return InvokeResponse{}, fmt.Errorf("failed to create socket dir: %w", err)
		}
		os.Chmod(socketDir, 0777)

		containerID, err := d.ContainerCreate(ctx, funcDir, socketDir)
		if err != nil {
			os.RemoveAll(socketDir)
			return InvokeResponse{}, fmt.Errorf("failed to create container: %w", err)
		}

		// Start the container
		if err := d.ContainerStart(ctx, containerID); err != nil {
			os.RemoveAll(socketDir)
			return InvokeResponse{}, fmt.Errorf("failed to start container: %w", err)
		}

		// Wait for the worker's UDS server to become ready
		workerSock := filepath.Join(socketDir, "glambdar.sock")
		if err := waitForSocket(workerSock, 5*time.Second); err != nil {
			d.ContainerKill(ctx, containerID)
			os.RemoveAll(socketDir)
			return InvokeResponse{}, fmt.Errorf("worker socket not ready: %w", err)
		}

		e = &pool.Entry{
			ContainerID:    containerID,
			SocketPath:     socketDir,
			ActiveRequests: 1, // this current request
		}
	}

	// Release container back to pool (or kill if pool is full) on return
	defer func() {
		if !p.Release(e) {
			d.ContainerKill(ctx, e.ContainerID)
			os.RemoveAll(e.SocketPath)
		}
	}()

	md.LastInvokedAt = time.Now().UTC()
	md.InvokeCount++
	if err := SaveMetadata(md); err != nil {
		log.Println("ERROR saving metadata:", err)
	}

	// Dial the container's UDS and send the request
	workerSock := filepath.Join(e.SocketPath, "glambdar.sock")
	conn, err := net.DialTimeout("unix", workerSock, 5*time.Second)
	if err != nil {
		return InvokeResponse{}, fmt.Errorf("failed to dial worker socket: %w", err)
	}
	defer conn.Close()

	// Encode request
	if err := json.NewEncoder(conn).Encode(req); err != nil {
		return InvokeResponse{}, fmt.Errorf("failed to send request: %w", err)
	}

	// Signal EOF so worker knows request is complete
	if uc, ok := conn.(*net.UnixConn); ok {
		uc.CloseWrite()
	}

	// Decode response
	var res InvokeResponse
	if err := json.NewDecoder(conn).Decode(&res); err != nil {
		return InvokeResponse{}, fmt.Errorf("failed to read response: %w", err)
	}

	// Process logs
	out, err := d.ContainerLogs(ctx, e.ContainerID, md.LastInvokedAt.Format(time.RFC3339))
	if err == nil {
		defer out.Close()
		var stdout, stderr bytes.Buffer
		stdcopy.StdCopy(&stdout, &stderr, out)

		// Save to DB
		SaveLog(&Log{
			FuncName:  funcName,
			InvokedAt: md.LastInvokedAt,
			Stdout:    stdout.String(),
			Stderr:    stderr.String(),
		})
	}

	res.ColdStart = !warm
	return res, nil
}

// waitForSocket polls until the unix socket at path is dial-able or timeout expires.
func waitForSocket(path string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		conn, err := net.DialTimeout("unix", path, 200*time.Millisecond)
		if err == nil {
			conn.Close()
			return nil
		}
		time.Sleep(50 * time.Millisecond)
	}
	return fmt.Errorf("timeout after %s waiting for %s", timeout, path)
}
