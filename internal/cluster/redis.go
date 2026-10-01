package cluster

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	NodeKeyPrefix = "glambdar:nodes:"
	WarmKeyPrefix = "glambdar:warm:"
	NodeTTL       = 45 * time.Second
)

type RedisStateProvider struct {
	client *redis.Client
}

func NewRedisStateProvider(redisAddr string) *RedisStateProvider {
	client := redis.NewClient(&redis.Options{
		Addr: redisAddr,
	})
	return &RedisStateProvider{client: client}
}

func (r *RedisStateProvider) PublishStatus(ctx context.Context, status *NodeStatus) error {
	status.LastBeat = time.Now()
	data, err := json.Marshal(status)
	if err != nil {
		return fmt.Errorf("failed to marshal node status: %w", err)
	}

	nodeKey := NodeKeyPrefix + status.NodeID
	pipe := r.client.Pipeline()
	pipe.Set(ctx, nodeKey, data, NodeTTL)

	for funcName, poolStatus := range status.Pools {
		warmKey := WarmKeyPrefix + funcName
		if poolStatus.IdleCount > 0 || poolStatus.ActiveCount > 0 {
			pipe.HSet(ctx, warmKey, status.NodeID, status.Address)
		} else {
			pipe.HDel(ctx, warmKey, status.NodeID)
		}
	}

	_, err = pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to execute pipeline for node status publish: %w", err)
	}

	return nil
}

func (r *RedisStateProvider) GetNodesWithWarmPool(ctx context.Context, funcName string) ([]NodeStatus, error) {
	warmKey := WarmKeyPrefix + funcName
	nodeIDs, err := r.client.HKeys(ctx, warmKey).Result()
	if err != nil && err != redis.Nil {
		return nil, fmt.Errorf("failed to get warm pool node IDs: %w", err)
	}

	if len(nodeIDs) == 0 {
		return nil, nil
	}

	var healthyWarmNodes []NodeStatus
	for _, nodeID := range nodeIDs {
		nodeKey := NodeKeyPrefix + nodeID
		val, err := r.client.Get(ctx, nodeKey).Result()
		if err == redis.Nil {
			r.client.HDel(ctx, warmKey, nodeID)
			continue
		} else if err != nil {
			return nil, fmt.Errorf("failed to fetch node status for %s: %w", nodeID, err)
		}

		var ns NodeStatus
		if err := json.Unmarshal([]byte(val), &ns); err != nil {
			continue
		}
		healthyWarmNodes = append(healthyWarmNodes, ns)
	}

	return healthyWarmNodes, nil
}

func (r *RedisStateProvider) GetHealthyNodes(ctx context.Context) ([]NodeStatus, error) {
	var cursor uint64
	var keys []string
	for {
		var k []string
		var err error
		k, cursor, err = r.client.Scan(ctx, cursor, NodeKeyPrefix+"*", 100).Result()
		if err != nil {
			return nil, fmt.Errorf("failed to scan healthy node keys: %w", err)
		}
		keys = append(keys, k...)
		if cursor == 0 {
			break
		}
	}

	if len(keys) == 0 {
		return nil, nil
	}

	vals, err := r.client.MGet(ctx, keys...).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to fetch healthy nodes data: %w", err)
	}

	var nodes []NodeStatus
	for _, val := range vals {
		if val == nil {
			continue
		}
		strVal, ok := val.(string)
		if !ok {
			continue
		}
		var ns NodeStatus
		if err := json.Unmarshal([]byte(strVal), &ns); err != nil {
			continue
		}
		nodes = append(nodes, ns)
	}

	return nodes, nil
}

func (r *RedisStateProvider) Join(ctx context.Context) error {
	return r.client.Ping(ctx).Err()
}

func (r *RedisStateProvider) Leave(ctx context.Context) error {
	return r.client.Close()
}
