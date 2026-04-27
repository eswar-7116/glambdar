package sockutil

import (
	"fmt"
	"net"
	"time"
)

// WaitForSocket polls until the unix socket at path is dial-able or timeout expires.
func WaitForSocket(path string, timeout time.Duration) error {
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
