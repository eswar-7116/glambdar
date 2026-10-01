package agent

import (
	"context"
	"runtime"
	"testing"

	"github.com/eswar-7116/glambdar/v3/internal/pool"
	"github.com/eswar-7116/glambdar/v3/proto"
)

func TestAgentServer_GetStatus_NodeID(t *testing.T) {
	pm := &pool.PoolManager{}
	srv := NewAgentServer("test-node-1", nil, pm, nil)

	resp, err := srv.GetStatus(context.Background(), &proto.StatusRequest{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.NodeId != "test-node-1" {
		t.Errorf("expected node ID 'test-node-1', got '%s'", resp.NodeId)
	}
}

func TestAgentServer_GetStatus_FallbackNodeID(t *testing.T) {
	pm := &pool.PoolManager{}
	srv := NewAgentServer("", nil, pm, nil)

	resp, err := srv.GetStatus(context.Background(), &proto.StatusRequest{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.NodeId == "" {
		t.Error("expected a non-empty fallback node ID")
	}
}

func TestAgentServer_GetStatus_Resources(t *testing.T) {
	pm := &pool.PoolManager{}
	srv := NewAgentServer("n1", nil, pm, nil)

	resp, err := srv.GetStatus(context.Background(), &proto.StatusRequest{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.CpuAvailable != float64(runtime.NumCPU()) {
		t.Errorf("expected CpuAvailable %d, got %f", runtime.NumCPU(), resp.CpuAvailable)
	}
	if resp.MemoryAvailable <= 0 {
		t.Errorf("expected positive MemoryAvailable, got %d", resp.MemoryAvailable)
	}
}

func TestAgentServer_GetStatus_EmptyPools(t *testing.T) {
	pm := &pool.PoolManager{}
	srv := NewAgentServer("n1", nil, pm, nil)

	resp, err := srv.GetStatus(context.Background(), &proto.StatusRequest{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Pools) != 0 {
		t.Errorf("expected no pools, got %d", len(resp.Pools))
	}
}

func TestAgentServer_PreloadFunction_EmptyName(t *testing.T) {
	srv := NewAgentServer("n1", nil, &pool.PoolManager{}, nil)

	resp, err := srv.PreloadFunction(context.Background(), &proto.PreloadRequest{FuncName: ""})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Success {
		t.Error("expected failure when func_name is empty")
	}
	if resp.Message == "" {
		t.Error("expected a non-empty message explaining the failure")
	}
}

func TestAgentServer_EvictFunction_EmptyName(t *testing.T) {
	srv := NewAgentServer("n1", nil, &pool.PoolManager{}, nil)

	resp, err := srv.EvictFunction(context.Background(), &proto.EvictRequest{FuncName: ""})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Success {
		t.Error("expected failure when func_name is empty")
	}
}

func TestAgentServer_EvictFunction_NonexistentFunc(t *testing.T) {
	srv := NewAgentServer("n1", nil, &pool.PoolManager{}, nil)

	resp, err := srv.EvictFunction(context.Background(), &proto.EvictRequest{FuncName: "nonexistent"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !resp.Success {
		t.Error("expected evict to succeed even for a non-existent function")
	}
}
