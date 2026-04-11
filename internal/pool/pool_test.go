package pool

import (
	"testing"
)

func TestContainerPool_AcquireRelease(t *testing.T) {
	p := &ContainerPool{
		idle: make(chan entry, 2),
	}

	// Try acquiring from empty pool
	id, sock, ok := p.Acquire()
	if ok || id != "" || sock != "" {
		t.Errorf("Expected pool to be empty, got id=%q, sock=%q, ok=%v", id, sock, ok)
	}

	// Release to pool
	ok = p.Release("container-1", "/tmp/glambdar-sock-1/")
	if !ok {
		t.Errorf("Expected release to succeed, pool not full")
	}

	ok = p.Release("container-2", "/tmp/glambdar-sock-2/")
	if !ok {
		t.Errorf("Expected release to succeed, pool not full")
	}

	// Try releasing when full
	ok = p.Release("container-3", "/tmp/glambdar-sock-3/")
	if ok {
		t.Errorf("Expected release to fail, pool is full")
	}

	// Acquire again from pool
	id, sock, ok = p.Acquire()
	if !ok {
		t.Errorf("Expected successful acquire, but it failed")
	}
	if (id != "container-1" && id != "container-2") || sock == "" {
		t.Errorf("Expected id to be container-1 or container-2 with a socket path, got id=%q sock=%q", id, sock)
	}
}
