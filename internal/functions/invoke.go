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

	// Accept UDS connection
	conn, err := l.AcceptUnix()
	if err != nil {
		return InvokeResponse{}, err
	}
	defer conn.Close()

	// Send request JSON via UDS
	if err := json.NewEncoder(conn).Encode(req); err != nil {
		return InvokeResponse{}, err
	}

	// Get response JSON via UDS
	var res InvokeResponse
	if err := json.NewDecoder(conn).Decode(&res); err != nil {
		return InvokeResponse{}, err
	}

	return res, nil
}
