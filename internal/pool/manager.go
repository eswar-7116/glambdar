package pool

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/eswar-7116/glambdar/internal/docker"
	"github.com/eswar-7116/glambdar/internal/ewma"
	"github.com/eswar-7116/glambdar/internal/sockutil"
	"golang.org/x/time/rate"
)

type PoolManager struct {
	pools sync.Map // funcName -> *ContainerPool
}

func (pm *PoolManager) GetOrCreate(funcName string, rateLimit int, maxConcurrency int32) (*ContainerPool, error) {
	if val, ok := pm.pools.Load(funcName); ok {
		return val.(*ContainerPool), nil
	}

	predictor, err := ewma.NewTrafficPredictor(0.2)
	if err != nil {
		return nil, fmt.Errorf("failed to create traffic predictor: %w", err)
	}

	p, _ := pm.pools.LoadOrStore(funcName, &ContainerPool{
		Idle:             make(chan *Entry, 10),
		Limiter:          newLimiter(rateLimit),
		MaxConcurrency:   maxConcurrency,
		TrafficPredictor: predictor,
	})
	return p.(*ContainerPool), nil
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

func (pm *PoolManager) DeletePool(ctx context.Context, d *docker.Docker, funcName string) {
	if val, ok := pm.pools.Load(funcName); ok {
		p := val.(*ContainerPool)
		// Drain the pool and remove containers
		for {
			select {
			case e := <-p.Idle:
				err := d.ContainerRemove(ctx, e.ContainerID)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Failed to delete container (%s) on pool deletion: %s\n", e.ContainerID[:12], err)
				}
				os.RemoveAll(e.SocketPath)
			default:
				pm.pools.Delete(funcName)
				return
			}
		}
	}
}

func (pm *PoolManager) DeleteAllContainers(ctx context.Context, d *docker.Docker) {
	pm.pools.Range(func(_, val any) bool {
		p := val.(*ContainerPool)
		for {
			select {
			case e := <-p.Idle:
				err := d.ContainerRemove(ctx, e.ContainerID)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Failed to delete container (%s): %s\n", e.ContainerID[:12], err)
				}
				os.RemoveAll(e.SocketPath)
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
			case e := <-p.Idle:
				if time.Since(e.LastUsed) > ttl {
					err := d.ContainerRemove(ctx, e.ContainerID)
					if err != nil {
						fmt.Fprintf(os.Stderr, "Failed to remove stale container (%s): %s\n", e.ContainerID[:12], err)
					}
					os.RemoveAll(e.SocketPath)
				} else {
					p.Idle <- e
					return true
				}
			default:
				return true // pool empty
			}
		}
	})
}

func (pm *PoolManager) StartPrewarmer(ctx context.Context, d *docker.Docker, functionsDir string, interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				pm.prewarm(ctx, d, functionsDir)
			case <-ctx.Done():
				return
			}
		}
	}()
}

func (pm *PoolManager) prewarm(ctx context.Context, d *docker.Docker, functionsDir string) {
	pm.pools.Range(func(key, val any) bool {
		funcName := key.(string)
		p := val.(*ContainerPool)

		count := p.InvokeCount.Swap(0)
		predicted := p.TrafficPredictor.Update(float64(count))

		idleNow := len(p.Idle)
		idleCap := cap(p.Idle)

		desired := int(predicted / 5)
		if desired < 1 {
			desired = 1
		}

		toSpawn := desired - idleNow
		for i := 0; i < toSpawn && idleNow+i < idleCap; i++ {
			go spawnIdle(ctx, d, functionsDir, funcName, p)
		}

		return true
	})
}

func spawnIdle(ctx context.Context, d *docker.Docker, functionsDir, funcName string, p *ContainerPool) {
	funcDir, err := filepath.Abs(filepath.Join(functionsDir, funcName))
	if err != nil {
		return
	}

	socketDir, err := os.MkdirTemp("", "glambdar-sock-*")
	if err != nil {
		return
	}
	os.Chmod(socketDir, 0777)

	containerID, err := d.ContainerCreate(ctx, funcDir, socketDir)
	if err != nil {
		os.RemoveAll(socketDir)
		return
	}
	if err := d.ContainerStart(ctx, containerID); err != nil {
		os.RemoveAll(socketDir)
		return
	}

	workerSock := filepath.Join(socketDir, "glambdar.sock")
	if err := sockutil.WaitForSocket(workerSock, 5*time.Second); err != nil {
		d.ContainerKill(ctx, containerID)
		os.RemoveAll(socketDir)
		return
	}

	entry := &Entry{
		ContainerID: containerID,
		SocketPath:  socketDir,
		LastUsed:    time.Now(),
	}

	select {
	case p.Idle <- entry:
		entry.InPool.Store(1)
	default:
		d.ContainerKill(ctx, containerID)
		os.RemoveAll(socketDir)
	}
}
