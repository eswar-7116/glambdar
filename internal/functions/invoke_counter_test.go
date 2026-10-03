package functions

import (
	"sync"
	"testing"
	"time"
)

func TestInvokeCounter_Increment(t *testing.T) {
	counter := &InvokeCounter{
		funcName: "test-func",
		interval: time.Hour, // long interval to avoid flush
	}
	before := time.Now().UTC()
	ts := counter.Increment()
	after := time.Now().UTC()

	if ts.Before(before) || ts.After(after) {
		t.Errorf("Increment timestamp %v outside expected range [%v, %v]", ts, before, after)
	}

	counter.mu.Lock()
	pending := counter.pending
	counter.mu.Unlock()
	if pending != 1 {
		t.Errorf("expected pending=1 after one Increment, got %d", pending)
	}
}

func TestInvokeCounter_MultipleIncrements(t *testing.T) {
	counter := &InvokeCounter{funcName: "test-func", interval: time.Hour}

	for range 10 {
		counter.Increment()
	}

	counter.mu.Lock()
	pending := counter.pending
	counter.mu.Unlock()
	if pending != 10 {
		t.Errorf("expected pending=10 after 10 Increments, got %d", pending)
	}
}

func TestInvokeCounter_FlushNoPending(t *testing.T) {
	counter := &InvokeCounter{funcName: "nonexistent-func", interval: time.Hour}

	counter.Flush()

	counter.mu.Lock()
	pending := counter.pending
	counter.mu.Unlock()
	if pending != 0 {
		t.Errorf("expected pending=0 after flush with no pending, got %d", pending)
	}
}

func TestInvokeCounter_CounterFor_SameInstance(t *testing.T) {
	// Reset
	invokeCounters = sync.Map{}
	defer func() { invokeCounters = sync.Map{} }()

	c1 := counterFor("same-func")
	c2 := counterFor("same-func")

	if c1 != c2 {
		t.Error("counterFor should return the same instance for the same funcName")
	}
}

func TestInvokeCounter_CounterFor_DifferentInstances(t *testing.T) {
	invokeCounters = sync.Map{}
	defer func() { invokeCounters = sync.Map{} }()

	c1 := counterFor("func-a")
	c2 := counterFor("func-b")

	if c1 == c2 {
		t.Error("counterFor should return different instances for different funcNames")
	}
}

func TestInvokeCounter_ConcurrentIncrement(t *testing.T) {
	counter := &InvokeCounter{funcName: "concurrent-func", interval: time.Hour}

	var wg sync.WaitGroup
	n := 100
	for range n {
		wg.Go(func() {
			counter.Increment()
		})
	}
	wg.Wait()

	counter.mu.Lock()
	pending := counter.pending
	counter.mu.Unlock()
	if pending != n {
		t.Errorf("expected pending=%d after %d concurrent Increments, got %d", n, n, pending)
	}
}

func TestInvokeCounter_Restore(t *testing.T) {
	counter := &InvokeCounter{funcName: "restore-func", interval: time.Hour}

	ts := time.Now().UTC()
	counter.restore(5, ts)

	counter.mu.Lock()
	pending := counter.pending
	lastInvoked := counter.lastInvokedAt
	counter.mu.Unlock()

	if pending != 5 {
		t.Errorf("expected pending=5 after restore, got %d", pending)
	}
	if !lastInvoked.Equal(ts) {
		t.Errorf("expected lastInvokedAt=%v after restore, got %v", ts, lastInvoked)
	}
}

func TestInvokeCounter_RestoreAccumulates(t *testing.T) {
	counter := &InvokeCounter{funcName: "restore-func", interval: time.Hour}

	ts1 := time.Now().UTC().Add(-time.Minute)
	ts2 := time.Now().UTC()

	counter.restore(3, ts1)
	counter.restore(7, ts2)

	counter.mu.Lock()
	pending := counter.pending
	lastInvoked := counter.lastInvokedAt
	counter.mu.Unlock()

	if pending != 10 {
		t.Errorf("expected pending=10 after two restores, got %d", pending)
	}
	// Should keep the later timestamp
	if !lastInvoked.Equal(ts2) {
		t.Errorf("expected lastInvokedAt=%v (later timestamp), got %v", ts2, lastInvoked)
	}
}

func TestFlushInvokeCounters_NoPanic(t *testing.T) {
	invokeCounters = sync.Map{}
	defer func() { invokeCounters = sync.Map{} }()

	// Empty map (should not panic)
	FlushInvokeCounters()
}
