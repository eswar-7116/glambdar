package pool

import (
	"sync"
	"testing"
)

func TestContainerPool_AcquireRelease_SingleConcurrency(t *testing.T) {
	p := &ContainerPool{
		Idle:           make(chan *Entry, 2),
		MaxConcurrency: 1,
	}

	// Try acquiring from empty pool
	e, ok := p.Acquire()
	if ok || e != nil {
		t.Errorf("Expected pool to be empty, got e=%v, ok=%v", e, ok)
	}

	// Create entry and add to pool
	e1 := &Entry{ContainerID: "c1", SocketPath: "s1"}
	e1.InPool.Store(1)
	p.Idle <- e1

	// Acquire
	e, ok = p.Acquire()
	if !ok || e != e1 {
		t.Errorf("Expected to acquire e1, got ok=%v", ok)
	}
	if e.ActiveRequests.Load() != 1 {
		t.Errorf("Expected 1 active request, got %d", e.ActiveRequests.Load())
	}

	// Check if pool is empty now (MaxConcurrency is 1)
	e2, ok := p.Acquire()
	if ok {
		t.Errorf("Expected pool to be empty (concurrency 1), got %v", e2)
	}

	// Release
	p.Release(e)
	if e.ActiveRequests.Load() != 0 {
		t.Errorf("Expected 0 active requests, got %d", e.ActiveRequests.Load())
	}

	// Acquire again
	e, ok = p.Acquire()
	if !ok || e != e1 {
		t.Errorf("Expected to acquire again")
	}
}

func TestContainerPool_HighConcurrency(t *testing.T) {
	p := &ContainerPool{
		Idle:           make(chan *Entry, 2),
		MaxConcurrency: 10,
	}

	e1 := &Entry{ContainerID: "c1", SocketPath: "s1"}
	e1.InPool.Store(1)
	p.Idle <- e1

	// Acquire multiple times
	for i := 1; i <= 10; i++ {
		e, ok := p.Acquire()
		if !ok || e != e1 {
			t.Fatalf("Failed to acquire at step %d", i)
		}
		if int(e.ActiveRequests.Load()) != i {
			t.Errorf("Expected %d active requests, got %d", i, e.ActiveRequests.Load())
		}

		// Pool should only be empty at the 11th call
		if i < 10 {
			// Check if it's still in the channel
			select {
			case entryInChan := <-p.Idle:
				if entryInChan != e1 {
					t.Errorf("Expected e1 in channel")
				}
				// Verify inPool flag - it should be 1 because Acquire should have put it back
				if entryInChan.InPool.Load() != 1 {
					t.Errorf("Expected InPool 1 because Acquire should have put it back")
				}
				// Put it back manually since we just popped it for verification
				p.Idle <- entryInChan
			default:
				t.Errorf("Expected entry to still be in channel at step %d", i)
			}
		}
	}

	// 11th call should fail
	_, ok := p.Acquire()
	if ok {
		t.Errorf("Expected pool to be empty at 11th call")
	}

	// Release one
	p.Release(e1)
	if e1.ActiveRequests.Load() != 9 {
		t.Errorf("Expected 9 active requests, got %d", e1.ActiveRequests.Load())
	}

	// Should be able to acquire again
	e, ok := p.Acquire()
	if !ok || e != e1 {
		t.Errorf("Expected to acquire again after release")
	}
}

func TestContainerPool_ConcurrentAccess(t *testing.T) {
	p := &ContainerPool{
		Idle:           make(chan *Entry, 1),
		MaxConcurrency: 100,
	}

	e1 := &Entry{ContainerID: "c1", SocketPath: "s1"}
	e1.InPool.Store(1)
	p.Idle <- e1

	var wg sync.WaitGroup
	for range 100 {
		wg.Go(func() {
			e, ok := p.Acquire()
			if ok && e != nil {
				// simulate work
				p.Release(e)
			}
		})
	}
	wg.Wait()

	if e1.ActiveRequests.Load() != 0 {
		t.Errorf("Expected 0 active requests after all finished, got %d", e1.ActiveRequests.Load())
	}
	if e1.InPool.Load() != 1 {
		t.Errorf("Expected InPool 1 after all released")
	}
}

func TestContainerPool_ReleaseNil(t *testing.T) {
	p := &ContainerPool{
		Idle:           make(chan *Entry, 2),
		MaxConcurrency: 10,
	}

	// Should return false
	if p.Release(nil) {
		t.Error("Expected Release(nil) to return false")
	}
}

func TestContainerPool_DefaultMaxConcurrency(t *testing.T) {
	p := &ContainerPool{
		Idle:           make(chan *Entry, 20),
		MaxConcurrency: 0, // Should default to 10
	}

	e1 := &Entry{ContainerID: "c1", SocketPath: "s1"}
	e1.InPool.Store(1)
	p.Idle <- e1

	e, ok := p.Acquire()
	if !ok || e == nil {
		t.Fatalf("Expected acquire to succeed")
	}
	if e.ActiveRequests.Load() != 1 {
		t.Errorf("Expected 1 active request, got %d", e.ActiveRequests.Load())
	}

	if len(p.Idle) != 1 {
		t.Errorf("Expected entry to be put back in pool, len is %d", len(p.Idle))
	}

	// Should decrement active requests
	p.Release(e)
	if e.ActiveRequests.Load() != 0 {
		t.Errorf("Expected 0 active requests, got %d", e.ActiveRequests.Load())
	}
}

func TestContainerPool_AcquireEmptyPoolReturnsFalse(t *testing.T) {
	p := &ContainerPool{
		Idle:           make(chan *Entry, 2),
		MaxConcurrency: 10,
	}

	e, ok := p.Acquire()
	if ok || e != nil {
		t.Errorf("Expected Acquire on empty pool to return (nil, false), got e=%v, ok=%v", e, ok)
	}
}
