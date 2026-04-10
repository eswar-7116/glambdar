package pool

import (
	"context"
	"fmt"
	"os"
	"sync"

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
