package functions

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"
	"time"

	"github.com/eswar-7116/glambdar/internal/config"
	"github.com/eswar-7116/glambdar/internal/docker"
)

type InvokeRequest struct {
	Headers map[string]string `json:"headers"`
	Body    string            `json:"body"`
}

type InvokeResponse struct {
	StatusCode int               `json:"statusCode"`
	Headers    map[string]string `json:"headers"`
	Body       json.RawMessage   `json:"body"`
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

	// Listen for connections in the UDS
	l, err := net.ListenUnix("unix", &net.UnixAddr{
		Name: config.UDSPath,
		Net:  "unix",
	})
	if err != nil {
		return InvokeResponse{}, err
	}
	defer l.Close()

	log.Println("Listening on /tmp/glambdar.sock...")
	fmt.Println(funcDir)
	fmt.Println(config.UDSPath)
	fmt.Println(config.WorkerPath)

	// // Invoke the function in a container
	// cmd := exec.Command(
	// 	"docker", "run",
	// 	"--rm", "-i",
	// 	"--memory=128m",
	// 	"--cpus=0.5",
	// 	"-v", funcDir+":/function",
	// 	"-v", util.UDSPath+":/glambdar/glambdar.sock",
	// 	"-v", util.WorkerPath+":/glambdar/worker.js",
	// 	"node:25-slim",
	// 	"node", "/glambdar/worker.js", "/function",
	// )
	// cmd.Stdout = os.Stdout
	// cmd.Stderr = os.Stderr
	// if err = cmd.Start(); err != nil {
	// 	return InvokeResponse{}, err
	// }

	// Create the container using your package
	p := config.PoolManager.GetOrCreate(funcName)
	containerID, warm := p.Acquire()
	if !warm {
		containerID, err = d.ContainerCreate(ctx, funcDir)
		if err != nil {
			return InvokeResponse{}, fmt.Errorf("failed to create container: %w", err)
		}
	}
	defer func() {
		if !p.Release(containerID) {
			d.ContainerKill(ctx, containerID)
		}
	}()

	// Start the container
	if err := d.ContainerStart(ctx, containerID); err != nil {
		return InvokeResponse{}, fmt.Errorf("failed to start container: %w", err)
	}

	// Update function metadata
	md, err := LoadMetadata(funcName)
	if err != nil {
		return InvokeResponse{}, err
	}
	md.LastInvokedAt = time.Now().UTC()
	md.InvokeCount++
	if err := SaveMetadata(md); err != nil {
		log.Println("ERROR saving metadata:", err)
	}

	connCh := make(chan *net.UnixConn, 1)

	// Accept UDS connection
	go func() {
		conn, err := l.AcceptUnix()
		if err == nil {
			connCh <- conn
		}
	}()

	select {
	case conn := <-connCh:
		defer conn.Close()

		// Encode request
		if err := json.NewEncoder(conn).Encode(req); err != nil {
			return InvokeResponse{}, err
		}

		// Decode response
		var res InvokeResponse
		if err := json.NewDecoder(conn).Decode(&res); err != nil {
			return InvokeResponse{}, err
		}

		return res, nil

	case <-time.After(5 * time.Second):
		err = d.ContainerKill(ctx, containerID)
		if err != nil {
			return InvokeResponse{}, err
		}
		return InvokeResponse{}, fmt.Errorf("timeout waiting for worker")
	}
}
