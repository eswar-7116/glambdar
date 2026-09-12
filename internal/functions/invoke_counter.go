package functions

import (
	"log"
	"sync"
	"time"
)

const invokeCounterFlushInterval = 30 * time.Second

type InvokeCounter struct {
	funcName string
	interval time.Duration

	mu            sync.Mutex
	pending       int
	lastInvokedAt time.Time
	startOnce     sync.Once
}

func NewInvokeCounter(funcName string, interval time.Duration) *InvokeCounter {
	counter := &InvokeCounter{funcName: funcName, interval: interval}
	counter.start()
	return counter
}

func (counter *InvokeCounter) start() {
	counter.startOnce.Do(func() {
		if counter.interval <= 0 {
			counter.interval = invokeCounterFlushInterval
		}
		go counter.run()
	})
}

func (counter *InvokeCounter) Increment() time.Time {
	now := time.Now().UTC()
	counter.mu.Lock()
	counter.pending++
	counter.lastInvokedAt = now
	counter.mu.Unlock()
	return now
}

func (counter *InvokeCounter) Flush() {
	counter.mu.Lock()
	pending := counter.pending
	lastInvokedAt := counter.lastInvokedAt
	counter.pending = 0
	counter.mu.Unlock()
	if pending == 0 {
		return
	}

	metadata, err := LoadMetadata(counter.funcName)
	if err != nil {
		counter.restore(pending, lastInvokedAt)
		log.Println("ERROR loading metadata:", err)
		return
	}
	metadata.LastInvokedAt = lastInvokedAt
	metadata.InvokeCount += pending
	if err := SaveMetadata(metadata); err != nil {
		counter.restore(pending, lastInvokedAt)
		log.Println("ERROR saving metadata:", err)
	}
}

func (counter *InvokeCounter) restore(pending int, lastInvokedAt time.Time) {
	counter.mu.Lock()
	counter.pending += pending
	if counter.lastInvokedAt.Before(lastInvokedAt) {
		counter.lastInvokedAt = lastInvokedAt
	}
	counter.mu.Unlock()
}

func (counter *InvokeCounter) run() {
	ticker := time.NewTicker(counter.interval)
	defer ticker.Stop()
	for range ticker.C {
		counter.Flush()
	}
}

var invokeCounters sync.Map

func counterFor(funcName string) *InvokeCounter {
	if counter, ok := invokeCounters.Load(funcName); ok {
		return counter.(*InvokeCounter)
	}
	counter := &InvokeCounter{funcName: funcName, interval: invokeCounterFlushInterval}
	actual, _ := invokeCounters.LoadOrStore(funcName, counter)
	if actual == counter {
		counter.start()
	}
	return actual.(*InvokeCounter)
}

func FlushInvokeCounters() {
	invokeCounters.Range(func(_, value any) bool {
		value.(*InvokeCounter).Flush()
		return true
	})
}
