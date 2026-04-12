package pool

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/eswar-7116/glambdar/internal/docker"
	"golang.org/x/time/rate"
)

type PoolManager struct {
	pools sync.Map // funcName -> *ContainerPool
}

func (pm *PoolManager) GetOrCreate(funcName string, rateLimit int) *ContainerPool {
	p, _ := pm.pools.LoadOrStore(funcName, &ContainerPool{
		idle:    make(chan entry, 10),
		Limiter: newLimiter(rateLimit),
	})
	return p.(*ContainerPool)
}

func (pm *PoolManager) UpdateLimiter(funcName string, rateLimit int) {
	if val, ok := pm.pools.Load(funcName); ok {
		p := val.(*ContainerPool)
		limit, burst := parseRateLimit(rateLimit)
		p.Limiter.SetLimit(limit)
		p.Limiter.SetBurst(burst)
	}
}

func newLimiter(rateLimit int) *rate.Limiter {
	limit, burst := parseRateLimit(rateLimit)
	return rate.NewLimiter(limit, burst)
}

func parseRateLimit(rateLimit int) (rate.Limit, int) {
	if rateLimit <= 0 {
		return rate.Inf, 1e9 // unlimited burst
	}
	burst := rateLimit / 10
	if burst < 1 {
		burst = 1
	}
	return rate.Limit(rateLimit), burst
}

func (pm *PoolManager) Delete(funcName string) {
	pm.pools.Delete(funcName)
}

func (pm *PoolManager) DeleteAllContainers(ctx context.Context, d *docker.Docker) {
	pm.pools.Range(func(_, val any) bool {
		p := val.(*ContainerPool)
		for {
			select {
			case e := <-p.idle:
				err := d.ContainerRemove(ctx, e.containerID)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Failed to delete container (%s): %s\n", e.containerID[:12], err)
				}
				os.RemoveAll(e.socketPath)
			default:
				return true // pool drained, move to next
			}
		}
	})
}

func (pm *PoolManager) RemoveStaleContainers(ctx context.Context, d *docker.Docker, ttl time.Duration) {
	pm.pools.Range(func(_, val any) bool {
		p := val.(*ContainerPool)
		for {
			select {
			case e := <-p.idle:
				if time.Since(e.lastUsed) > ttl {
					err := d.ContainerRemove(ctx, e.containerID)
					if err != nil {
						fmt.Fprintf(os.Stderr, "Failed to remove stale container (%s): %s\n", e.containerID[:12], err)
					}
					os.RemoveAll(e.socketPath)
				} else {
					p.idle <- e
					return true
				}
			default:
				return true // pool empty
			}
		}
	})
}
