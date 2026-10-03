package ratelimit

import (
	"context"
	_ "embed"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
)

const keyPrefix = "glambdar:ratelimit:"

//go:embed token_bucket.lua
var luaTokenBucketSource string
var luaTokenBucket = redis.NewScript(luaTokenBucketSource)

type RedisLimiter struct {
	client *redis.Client
}

func NewRedisLimiter(client *redis.Client) *RedisLimiter {
	return &RedisLimiter{client: client}
}

func (r *RedisLimiter) Allow(ctx context.Context, funcName string, limit int) (bool, error) {
	// Unlimited tokens check
	if limit <= 0 {
		return true, nil
	}

	key := keyPrefix + funcName
	nowMs := time.Now().UnixMilli()

	result, err := luaTokenBucket.Run(ctx, r.client, []string{key}, limit, limit, nowMs, 1).Int()

	if err != nil {
		// Fail open
		log.Printf("[ratelimit] Redis error, failing open: %v", err)
		return true, nil
	}

	return result == 1, nil
}
