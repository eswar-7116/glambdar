package ratelimit

import (
	"context"
	"sync"

	"golang.org/x/time/rate"
)

type LocalLimiter struct {
	mu       sync.RWMutex
	limiters map[string]*rate.Limiter
}

func NewLocalLimiter() *LocalLimiter {
	return &LocalLimiter{limiters: make(map[string]*rate.Limiter)}
}

func (l *LocalLimiter) Allow(_ context.Context, funcName string, limit int) (bool, error) {
	if limit <= 0 {
		return true, nil
	}

	l.mu.RLock()
	lim, ok := l.limiters[funcName]
	l.mu.RUnlock()
	if !ok {
		l.mu.Lock()
		lim, ok = l.limiters[funcName]
		if !ok {
			burst := max(limit/10, 1)
			lim = rate.NewLimiter(rate.Limit(limit), burst)
			l.limiters[funcName] = lim
		}
		l.mu.Unlock()
	}
	return lim.Allow(), nil
}

func (l *LocalLimiter) UpdateLimit(funcName string, limit int) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if lim, ok := l.limiters[funcName]; ok {
		if limit <= 0 {
			lim.SetLimit(rate.Inf)
			lim.SetBurst(1e9)
		} else {
			lim.SetLimit(rate.Limit(limit))
			lim.SetBurst(max(limit/10, 1))
		}
	}
}
