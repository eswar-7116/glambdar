package pool

import (
	"sync/atomic"
	"time"

	"github.com/eswar-7116/glambdar/internal/ewma"
	"golang.org/x/time/rate"
)

type Entry struct {
	ContainerID    string
	SocketPath     string
	LastUsed       time.Time
	ActiveRequests int32
	InPool         int32 // atomic bool: 1 if in Idle channel, 0 otherwise
}

type ContainerPool struct {
	Idle             chan *Entry
	Limiter          *rate.Limiter
	MaxConcurrency   int32
	InvokeCount      atomic.Int64
	TrafficPredictor *ewma.TrafficPredictor
}

func (p *ContainerPool) Acquire() (entry *Entry, warm bool) {
	if p.MaxConcurrency <= 0 {
		p.MaxConcurrency = 10
	}

	select {
	case e := <-p.Idle:
		atomic.StoreInt32(&e.InPool, 0)
		active := atomic.AddInt32(&e.ActiveRequests, 1)
		if active < p.MaxConcurrency {
			// Still has capacity, try to put back in pool
			if atomic.CompareAndSwapInt32(&e.InPool, 0, 1) {
				select {
				case p.Idle <- e:
				default:
					atomic.StoreInt32(&e.InPool, 0)
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
	newActive := atomic.AddInt32(&e.ActiveRequests, -1)
	e.LastUsed = time.Now()

	if newActive < p.MaxConcurrency {
		// Try to put back in pool if not already there
		if atomic.CompareAndSwapInt32(&e.InPool, 0, 1) {
			select {
			case p.Idle <- e:
				return true
			default:
				atomic.StoreInt32(&e.InPool, 0)
				return false // Pool full, caller should kill
			}
		}
	}

	return true
}
