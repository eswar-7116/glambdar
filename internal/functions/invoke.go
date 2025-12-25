package functions

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/eswar-7116/glambdar/internal/util"
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

func Invoke(funcName string, req InvokeRequest) (InvokeResponse, error) {
	funcDir, err := filepath.Abs(filepath.Join(util.FunctionsDir, funcName))
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
		Name: util.UDSPath,
		Net:  "unix",
	})
	if err != nil {
		return InvokeResponse{}, err
	}
	defer l.Close()

	log.Println("Listening on /tmp/glambdar.sock...")

	// Invoke the function in a container
	cmd := exec.Command(
		"docker", "run",
		"--rm", "-i",
		"--memory=128m",
		"--cpus=0.5",
		"-v", funcDir+":/function",
		"-v", util.UDSPath+":/glambdar/glambdar.sock",
		"-v", util.WorkerPath+":/glambdar/worker.js",
		"node:25-slim",
		"node", "/glambdar/worker.js", "/function",
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err = cmd.Start(); err != nil {
		return InvokeResponse{}, err
	}

	// Update function metadata
	md, err := LoadMetadata(funcDir)
	if err != nil {
		return InvokeResponse{}, err
	}
	md.LastInvokedAt = time.Now().UTC()
	md.InvokeCount++
	SaveMetadata(funcDir, md)

	connCh := make(chan *net.UnixConn, 1)
	procCh := make(chan error, 1)

	// Accept UDS connection
	go func() {
		conn, err := l.AcceptUnix()
		if err == nil {
			connCh <- conn
		}
	}()

	go func() {
		procCh <- cmd.Wait()
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

	case err := <-procCh:
		return InvokeResponse{}, fmt.Errorf("worker exited early: %v", err)

	case <-time.After(5 * time.Second):
		_ = cmd.Process.Kill()
		return InvokeResponse{}, fmt.Errorf("timeout waiting for worker (is Docker running?)")
	}
}
