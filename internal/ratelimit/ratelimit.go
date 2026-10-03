package ratelimit

import "context"

type Limiter interface {
	// limit <= 0 means "unlimited"
	Allow(ctx context.Context, funcName string, limit int) (bool, error)
}
