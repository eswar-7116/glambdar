package cluster

import (
	"context"
	"testing"
	"time"
)

type MockStateProvider struct {
	WarmNodes    map[string][]NodeStatus
	HealthyNodes []NodeStatus
}

func (m *MockStateProvider) PublishStatus(ctx context.Context, status *NodeStatus) error {
	return nil
}

func (m *MockStateProvider) GetNodesWithWarmPool(ctx context.Context, funcName string) ([]NodeStatus, error) {
	return m.WarmNodes[funcName], nil
}

func (m *MockStateProvider) GetHealthyNodes(ctx context.Context) ([]NodeStatus, error) {
	return m.HealthyNodes, nil
}

func (m *MockStateProvider) Join(ctx context.Context) error {
	return nil
}

func (m *MockStateProvider) Leave(ctx context.Context) error {
	return nil
}

func TestRouter_WarmPoolAffinity(t *testing.T) {
	mock := &MockStateProvider{
		WarmNodes: map[string][]NodeStatus{
			"fn1": {
				{
					NodeID:  "node-1",
					Address: "localhost:9091",
					Pools: map[string]PoolStatus{
						"fn1": {IdleCount: 2},
					},
				},
				{
					NodeID:  "node-2",
					Address: "localhost:9092",
					Pools: map[string]PoolStatus{
						"fn1": {IdleCount: 5},
					},
				},
			},
		},
	}

	router := NewRouter(mock)
	selected, err := router.RouteInvocation(context.Background(), "fn1")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if selected.NodeID != "node-2" {
		t.Errorf("expected node-2 (highest idle count), got %s", selected.NodeID)
	}
}

func TestRouter_ColdStartFallback(t *testing.T) {
	mock := &MockStateProvider{
		WarmNodes: map[string][]NodeStatus{},
		HealthyNodes: []NodeStatus{
			{
				NodeID:  "node-1",
				Address: "localhost:9091",
				Capacity: ResourceCapacity{
					MemoryAvailable: 1024,
					CPUAvailable:    2.0,
				},
			},
			{
				NodeID:  "node-2",
				Address: "localhost:9092",
				Capacity: ResourceCapacity{
					MemoryAvailable: 2048,
					CPUAvailable:    4.0,
				},
			},
		},
	}

	router := NewRouter(mock)
	selected, err := router.RouteInvocation(context.Background(), "fn2")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if selected.NodeID != "node-2" {
		t.Errorf("expected node-2 (highest memory capacity), got %s", selected.NodeID)
	}
}

func TestRouter_RoundRobinTieBreak(t *testing.T) {
	mock := &MockStateProvider{
		WarmNodes: map[string][]NodeStatus{
			"fn1": {
				{
					NodeID:  "node-1",
					Address: "localhost:9091",
					Pools: map[string]PoolStatus{
						"fn1": {IdleCount: 3},
					},
				},
				{
					NodeID:  "node-2",
					Address: "localhost:9092",
					Pools: map[string]PoolStatus{
						"fn1": {IdleCount: 3},
					},
				},
			},
		},
	}

	router := NewRouter(mock)
	n1, err := router.RouteInvocation(context.Background(), "fn1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	n2, err := router.RouteInvocation(context.Background(), "fn1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if n1.NodeID == n2.NodeID {
		t.Errorf("expected round robin distribution between tied nodes, got same node: %s", n1.NodeID)
	}
}

func TestRouter_NoNodes(t *testing.T) {
	mock := &MockStateProvider{
		WarmNodes:    map[string][]NodeStatus{},
		HealthyNodes: []NodeStatus{},
	}

	router := NewRouter(mock)
	_, err := router.RouteInvocation(context.Background(), "fn1")
	if err != ErrNoNodesAvailable {
		t.Errorf("expected ErrNoNodesAvailable, got %v", err)
	}
}

func TestNodeStatus_Marshaling(t *testing.T) {
	ns := NodeStatus{
		NodeID:   "node-test",
		Address:  "127.0.0.1:9090",
		LastBeat: time.Now(),
		Pools: map[string]PoolStatus{
			"test-fn": {IdleCount: 1, ActiveCount: 0, MaxConcurrency: 10},
		},
		Capacity: ResourceCapacity{CPUAvailable: 4.0, MemoryAvailable: 8192},
	}
	if ns.NodeID != "node-test" {
		t.Errorf("unexpected node ID: %s", ns.NodeID)
	}
}

func TestRouter_SingleWarmNode(t *testing.T) {
	mock := &MockStateProvider{
		WarmNodes: map[string][]NodeStatus{
			"fn1": {
				{NodeID: "node-1", Address: "localhost:9091", Pools: map[string]PoolStatus{"fn1": {IdleCount: 1}}},
			},
		},
	}
	router := NewRouter(mock)
	selected, err := router.RouteInvocation(context.Background(), "fn1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if selected.NodeID != "node-1" {
		t.Errorf("expected node-1, got %s", selected.NodeID)
	}
}

func TestRouter_WarmNodesMixedIdleCounts(t *testing.T) {
	mock := &MockStateProvider{
		WarmNodes: map[string][]NodeStatus{
			"fn1": {
				{NodeID: "node-1", Address: "localhost:9091", Pools: map[string]PoolStatus{"fn1": {IdleCount: 1}}},
				{NodeID: "node-2", Address: "localhost:9092", Pools: map[string]PoolStatus{"fn1": {IdleCount: 5}}},
				{NodeID: "node-3", Address: "localhost:9093", Pools: map[string]PoolStatus{"fn1": {IdleCount: 3}}},
			},
		},
	}
	router := NewRouter(mock)
	selected, err := router.RouteInvocation(context.Background(), "fn1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if selected.NodeID != "node-2" {
		t.Errorf("expected node-2 (idle count 5), got %s", selected.NodeID)
	}
}

func TestRouter_ColdStart_TiedCapacity(t *testing.T) {
	mock := &MockStateProvider{
		WarmNodes: map[string][]NodeStatus{},
		HealthyNodes: []NodeStatus{
			{NodeID: "node-1", Address: "localhost:9091", Capacity: ResourceCapacity{MemoryAvailable: 1024, CPUAvailable: 2.0}},
			{NodeID: "node-2", Address: "localhost:9092", Capacity: ResourceCapacity{MemoryAvailable: 1024, CPUAvailable: 2.0}},
		},
	}
	router := NewRouter(mock)
	n1, _ := router.RouteInvocation(context.Background(), "fn1")
	n2, _ := router.RouteInvocation(context.Background(), "fn1")
	if n1.NodeID == n2.NodeID {
		t.Errorf("expected round robin distribution between tied cold start nodes")
	}
}

func TestRouter_ColdStart_CPUTiebreaker(t *testing.T) {
	mock := &MockStateProvider{
		WarmNodes: map[string][]NodeStatus{},
		HealthyNodes: []NodeStatus{
			{NodeID: "node-1", Address: "localhost:9091", Capacity: ResourceCapacity{MemoryAvailable: 1024, CPUAvailable: 2.0}},
			{NodeID: "node-2", Address: "localhost:9092", Capacity: ResourceCapacity{MemoryAvailable: 1024, CPUAvailable: 4.0}},
		},
	}
	router := NewRouter(mock)
	selected, err := router.RouteInvocation(context.Background(), "fn1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if selected.NodeID != "node-2" {
		t.Errorf("expected node-2 (higher CPU with same memory), got %s", selected.NodeID)
	}
}

func TestRouter_RoundRobin_Wraps(t *testing.T) {
	mock := &MockStateProvider{
		WarmNodes: map[string][]NodeStatus{
			"fn1": {
				{NodeID: "node-1", Address: "localhost:9091", Pools: map[string]PoolStatus{"fn1": {IdleCount: 1}}},
				{NodeID: "node-2", Address: "localhost:9092", Pools: map[string]PoolStatus{"fn1": {IdleCount: 1}}},
			},
		},
	}
	router := NewRouter(mock)

	// Simulate many requests
	counts := make(map[string]int)
	for range 100 {
		selected, _ := router.RouteInvocation(context.Background(), "fn1")
		counts[selected.NodeID]++
	}

	if counts["node-1"] != 50 || counts["node-2"] != 50 {
		t.Errorf("expected 50/50 split, got %d/%d", counts["node-1"], counts["node-2"])
	}
}

type errorStateProvider struct{}

func (e *errorStateProvider) PublishStatus(ctx context.Context, status *NodeStatus) error {
	return nil
}
func (e *errorStateProvider) GetNodesWithWarmPool(ctx context.Context, funcName string) ([]NodeStatus, error) {
	return nil, context.DeadlineExceeded
}
func (e *errorStateProvider) GetHealthyNodes(ctx context.Context) ([]NodeStatus, error) {
	return nil, context.DeadlineExceeded
}
func (e *errorStateProvider) Join(ctx context.Context) error {
	return nil
}
func (e *errorStateProvider) Leave(ctx context.Context) error {
	return nil
}

func TestRouter_StateProviderError(t *testing.T) {
	router := NewRouter(&errorStateProvider{})
	_, err := router.RouteInvocation(context.Background(), "fn1")
	if err != context.DeadlineExceeded {
		t.Errorf("expected DeadlineExceeded error, got %v", err)
	}
}

type halfErrorStateProvider struct{}

func (e *halfErrorStateProvider) PublishStatus(ctx context.Context, status *NodeStatus) error {
	return nil
}
func (e *halfErrorStateProvider) GetNodesWithWarmPool(ctx context.Context, funcName string) ([]NodeStatus, error) {
	return nil, nil
}
func (e *halfErrorStateProvider) GetHealthyNodes(ctx context.Context) ([]NodeStatus, error) {
	return nil, context.DeadlineExceeded
}
func (e *halfErrorStateProvider) Join(ctx context.Context) error {
	return nil
}
func (e *halfErrorStateProvider) Leave(ctx context.Context) error {
	return nil
}

func TestRouter_HealthyNodesError(t *testing.T) {
	router := NewRouter(&halfErrorStateProvider{})
	_, err := router.RouteInvocation(context.Background(), "fn1")
	if err != context.DeadlineExceeded {
		t.Errorf("expected DeadlineExceeded error, got %v", err)
	}
}
