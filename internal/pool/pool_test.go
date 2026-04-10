package pool

import (
	"testing"
)

func TestContainerPool_AcquireRelease(t *testing.T) {
	p := &ContainerPool{
		idle: make(chan entry, 2),
	}

	// Try acquiring from empty pool
	id, ok := p.Acquire()
	if ok || id != "" {
		t.Errorf("Expected pool to be empty, got id=%q, ok=%v", id, ok)
	}

	// Release to pool
	ok = p.Release("container-1")
	if !ok {
		t.Errorf("Expected release to succeed, pool not full")
	}

	ok = p.Release("container-2")
	if !ok {
		t.Errorf("Expected release to succeed, pool not full")
	}

	// Try releasing when full
	ok = p.Release("container-3")
	if ok {
		t.Errorf("Expected release to fail, pool is full")
	}

	// Acquire again from pool
	id, ok = p.Acquire()
	if !ok {
		t.Errorf("Expected successful acquire, but it failed")
	}
	if id != "container-1" && id != "container-2" {
		t.Errorf("Expected id to be container-1 or container-2, got %q", id)
	}
}
