package pool

import (
	"sync"
	"sync/atomic"
	"time"

	"github.com/eswar-7116/glambdar/v3/internal/ewma"
	"golang.org/x/time/rate"
)

type Entry struct {
	ContainerID    string
	SocketPath     string
	mu             sync.Mutex
	LastUsed       time.Time
	ActiveRequests atomic.Int32
	InPool         atomic.Int32 // atomic bool: 1 if in Idle channel, 0 otherwise
}

type ContainerPool struct {
	Idle             chan *Entry
	Limiter          *rate.Limiter
	MaxConcurrency   int32
	InvokeCount      atomic.Int64
	TrafficPredictor *ewma.TrafficPredictor
}

func (p *ContainerPool) Acquire() (entry *Entry, warm bool) {
	limit := p.MaxConcurrency
	if limit <= 0 {
		limit = 10
	}

	select {
	case e := <-p.Idle:
		e.InPool.Store(0)
		active := e.ActiveRequests.Add(1)
		if active < limit {
			// Still has capacity, try to put back in pool
			if e.InPool.CompareAndSwap(0, 1) {
				select {
				case p.Idle <- e:
				default:
					e.InPool.Store(0)
				}
			}
		}
		return e, true
	default:
		return nil, false
	}
}

func (p *ContainerPool) Release(e *Entry) bool {
	if e == nil {
		return false
	}
	newActive := e.ActiveRequests.Add(-1)
	e.mu.Lock()
	e.LastUsed = time.Now()
	e.mu.Unlock()

	limit := p.MaxConcurrency
	if limit <= 0 {
		limit = 10
	}

	if newActive < limit {
		// Try to put back in pool if not already there
		if e.InPool.CompareAndSwap(0, 1) {
			select {
			case p.Idle <- e:
				return true
			default:
				e.InPool.Store(0)
				return false // Pool full, caller should kill
			}
		}
	}

	return true
}
