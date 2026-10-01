package cluster

import (
	"context"
	"errors"
	"sort"
	"sync/atomic"
)

var ErrNoNodesAvailable = errors.New("no healthy agent nodes available in cluster")

type Router struct {
	provider StateProvider
	rrIdx    atomic.Uint64
}

func NewRouter(provider StateProvider) *Router {
	return &Router{
		provider: provider,
	}
}

// Round-robin among top warm nodes
func (r *Router) RouteInvocation(ctx context.Context, funcName string) (*NodeStatus, error) {
	warmNodes, err := r.provider.GetNodesWithWarmPool(ctx, funcName)
	if err != nil {
		return nil, err
	}

	if len(warmNodes) > 0 {
		sort.Slice(warmNodes, func(i, j int) bool {
			return warmNodes[i].Pools[funcName].IdleCount > warmNodes[j].Pools[funcName].IdleCount
		})

		maxIdle := warmNodes[0].Pools[funcName].IdleCount
		var candidates []NodeStatus
		for _, n := range warmNodes {
			if n.Pools[funcName].IdleCount == maxIdle {
				candidates = append(candidates, n)
			} else {
				break
			}
		}

		idx := r.rrIdx.Add(1)
		selected := candidates[int(idx-1)%len(candidates)]
		return &selected, nil
	}

	healthyNodes, err := r.provider.GetHealthyNodes(ctx)
	if err != nil {
		return nil, err
	}

	if len(healthyNodes) == 0 {
		return nil, ErrNoNodesAvailable
	}

	sort.Slice(healthyNodes, func(i, j int) bool {
		if healthyNodes[i].Capacity.MemoryAvailable == healthyNodes[j].Capacity.MemoryAvailable {
			return healthyNodes[i].Capacity.CPUAvailable > healthyNodes[j].Capacity.CPUAvailable
		}
		return healthyNodes[i].Capacity.MemoryAvailable > healthyNodes[j].Capacity.MemoryAvailable
	})

	maxMem := healthyNodes[0].Capacity.MemoryAvailable
	maxCPU := healthyNodes[0].Capacity.CPUAvailable
	var candidates []NodeStatus
	for _, n := range healthyNodes {
		if n.Capacity.MemoryAvailable == maxMem && n.Capacity.CPUAvailable == maxCPU {
			candidates = append(candidates, n)
		} else {
			break
		}
	}

	idx := r.rrIdx.Add(1)
	selected := candidates[int(idx-1)%len(candidates)]
	return &selected, nil
}
