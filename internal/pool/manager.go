package pool

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/eswar-7116/glambdar/v3/internal/docker"
	"github.com/eswar-7116/glambdar/v3/internal/ewma"
	"github.com/eswar-7116/glambdar/v3/internal/sockutil"
)

type PoolManager struct {
	pools sync.Map // funcName -> *ContainerPool
}

type PoolStatus struct {
	IdleCount      int32
	ActiveCount    int32
	MaxConcurrency int32
}

func (pm *PoolManager) GetPoolStatuses() map[string]PoolStatus {
	statuses := make(map[string]PoolStatus)
	pm.pools.Range(func(key, val any) bool {
		funcName := key.(string)
		p := val.(*ContainerPool)
		statuses[funcName] = PoolStatus{
			IdleCount:      int32(len(p.Idle)),
			ActiveCount:    0,
			MaxConcurrency: p.MaxConcurrency,
		}
		return true
	})
	return statuses
}

func (pm *PoolManager) GetOrCreate(funcName string, maxConcurrency int32) (*ContainerPool, error) {
	if val, ok := pm.pools.Load(funcName); ok {
		return val.(*ContainerPool), nil
	}

	predictor, err := ewma.NewTrafficPredictor(0.2)
	if err != nil {
		return nil, fmt.Errorf("failed to create traffic predictor: %w", err)
	}

	p, _ := pm.pools.LoadOrStore(funcName, &ContainerPool{
		Idle:             make(chan *Entry, 10),
		MaxConcurrency:   maxConcurrency,
		TrafficPredictor: predictor,
	})
	return p.(*ContainerPool), nil
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
				e.mu.Lock()
				lastUsed := e.LastUsed
				e.mu.Unlock()
				if time.Since(lastUsed) > ttl {
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

		desired := max(int(predicted/5), 1)

		toSpawn := desired - idleNow
		for i := 0; i < toSpawn && idleNow+i < idleCap; i++ {
			go SpawnIdle(ctx, d, functionsDir, funcName, p)
		}

		return true
	})
}

func SpawnIdle(ctx context.Context, d *docker.Docker, functionsDir, funcName string, p *ContainerPool) {
	funcDir, err := filepath.Abs(filepath.Join(functionsDir, funcName))
	if err != nil {
		return
	}

	socketDir, err := os.MkdirTemp("", "glambdar-sock-*")
	if err != nil {
		return
	}
	if err := os.Chmod(socketDir, 0777); err != nil {
		os.RemoveAll(socketDir)
		return
	}

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
