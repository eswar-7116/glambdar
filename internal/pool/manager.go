package pool

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/eswar-7116/glambdar/internal/docker"
)

type PoolManager struct {
	pools sync.Map // funcName -> *ContainerPool
}

func (pm *PoolManager) GetOrCreate(funcName string) *ContainerPool {
	p, _ := pm.pools.LoadOrStore(funcName, &ContainerPool{
		idle: make(chan entry, 10),
	})
	return p.(*ContainerPool)
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
