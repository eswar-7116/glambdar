package sockutil_test

import (
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/eswar-7116/glambdar/internal/sockutil"
)

func TestWaitForSocket_Success(t *testing.T) {
	tmpDir := t.TempDir()
	socketPath := filepath.Join(tmpDir, "test_success.sock")

	// Start listening on the unix socket
	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		t.Fatalf("Failed to listen on unix socket: %v", err)
	}
	defer listener.Close()

	err = sockutil.WaitForSocket(socketPath, 1*time.Second)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
}

func TestWaitForSocket_Timeout(t *testing.T) {
	tmpDir := t.TempDir()
	socketPath := filepath.Join(tmpDir, "test_timeout.sock")

	// Call WaitForSocket on a non-existent socket path
	startTime := time.Now()
	timeout := 150 * time.Millisecond
	err := sockutil.WaitForSocket(socketPath, timeout)
	duration := time.Since(startTime)

	if err == nil {
		t.Fatalf("Expected error, got nil")
	}

	if !strings.Contains(err.Error(), "timeout after") {
		t.Errorf("Expected timeout error message, got: %v", err)
	}

	// Timeout must occur
	if duration < timeout {
		t.Errorf("WaitForSocket returned too early: got %v, expected at least %v", duration, timeout)
	}
}

func TestWaitForSocket_Delay(t *testing.T) {
	tmpDir := t.TempDir()
	socketPath := filepath.Join(tmpDir, "test_delay.sock")

	// Create the listener after a short delay
	go func() {
		time.Sleep(100 * time.Millisecond)
		listener, err := net.Listen("unix", socketPath)
		if err == nil {
			defer listener.Close()
			time.Sleep(200 * time.Millisecond)
		}
	}()

	defer os.Remove(socketPath)

	err := sockutil.WaitForSocket(socketPath, 1*time.Second)
	if err != nil {
		t.Fatalf("Expected socket to eventually become ready, but got error: %v", err)
	}
}
