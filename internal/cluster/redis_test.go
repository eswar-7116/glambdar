package cluster

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
)

func requireTestRedis(t *testing.T) string {
	t.Helper()
	if os.Getenv("RUN_INTEGRATION_TESTS") != "1" {
		t.Skip("RUN_INTEGRATION_TESTS != 1; skipping integration tests")
	}
	addr := os.Getenv("TEST_REDIS_ADDR")
	if addr == "" {
		t.Skip("TEST_REDIS_ADDR not set; skipping Redis test (set TEST_REDIS_ADDR=localhost:6379 to run)")
	}

	client := redis.NewClient(&redis.Options{Addr: addr})
	if err := client.Ping(context.Background()).Err(); err != nil {
		t.Skipf("Cannot connect to Redis at %s: %v", addr, err)
	}

	return addr
}

func TestRedisStateProvider_PublishAndGetHealthy(t *testing.T) {
	addr := requireTestRedis(t)
	r := NewRedisStateProvider(addr)
	ctx := context.Background()

	node := &NodeStatus{
		NodeID:  "node-1",
		Address: "127.0.0.1:9090",
		Capacity: ResourceCapacity{
			CPUAvailable:    2.0,
			MemoryAvailable: 1024,
		},
	}

	if err := r.PublishStatus(ctx, node); err != nil {
		t.Fatalf("PublishStatus failed: %v", err)
	}

	nodes, err := r.GetHealthyNodes(ctx)
	if err != nil {
		t.Fatalf("GetHealthyNodes failed: %v", err)
	}

	found := false
	for _, n := range nodes {
		if n.NodeID == "node-1" {
			found = true
			if n.Address != "127.0.0.1:9090" {
				t.Errorf("Expected address 127.0.0.1:9090, got %s", n.Address)
			}
			break
		}
	}
	if !found {
		t.Error("Published node not found in healthy nodes")
	}
}

func TestRedisStateProvider_PublishWarmPool(t *testing.T) {
	addr := requireTestRedis(t)
	r := NewRedisStateProvider(addr)
	ctx := context.Background()

	node := &NodeStatus{
		NodeID:  "node-warm",
		Address: "127.0.0.1:9091",
		Pools: map[string]PoolStatus{
			"warm-fn": {IdleCount: 5, ActiveCount: 2, MaxConcurrency: 10},
		},
	}

	if err := r.PublishStatus(ctx, node); err != nil {
		t.Fatalf("PublishStatus failed: %v", err)
	}

	nodes, err := r.GetNodesWithWarmPool(ctx, "warm-fn")
	if err != nil {
		t.Fatalf("GetNodesWithWarmPool failed: %v", err)
	}

	found := false
	for _, n := range nodes {
		if n.NodeID == "node-warm" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Published warm node not found in warm pool index")
	}
}

func TestRedisStateProvider_JoinLeave(t *testing.T) {
	addr := requireTestRedis(t)
	r := NewRedisStateProvider(addr)
	ctx := context.Background()

	if err := r.Join(ctx); err != nil {
		t.Errorf("Join failed: %v", err)
	}

	if err := r.Leave(ctx); err != nil {
		t.Errorf("Leave failed: %v", err)
	}

	// Should fail (client is closed)
	if err := r.Join(ctx); err == nil {
		t.Error("Expected Join to fail after Leave")
	}
}

func TestRedisStateProvider_EmptyCluster(t *testing.T) {
	addr := requireTestRedis(t)
	r := NewRedisStateProvider(addr)
	ctx := context.Background()

	nodes, err := r.GetNodesWithWarmPool(ctx, "nonexistent-fn-"+time.Now().String())
	if err != nil {
		t.Fatalf("GetNodesWithWarmPool failed: %v", err)
	}
	if len(nodes) != 0 {
		t.Errorf("Expected 0 nodes, got %d", len(nodes))
	}
}
