package agent

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/eswar-7116/glambdar/v3/internal/config"
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

func TestAgentServer_GetStatus_WithPools(t *testing.T) {
	pm := &pool.PoolManager{}
	pm.GetOrCreate("func-1", 10)
	pm.GetOrCreate("func-2", 20)

	srv := NewAgentServer("n1", nil, pm, nil)

	resp, err := srv.GetStatus(context.Background(), &proto.StatusRequest{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(resp.Pools) != 2 {
		t.Fatalf("expected 2 pools in status, got %d", len(resp.Pools))
	}

	if resp.Pools["func-1"].MaxConcurrency != 10 {
		t.Errorf("expected func-1 MaxConcurrency 10, got %d", resp.Pools["func-1"].MaxConcurrency)
	}
	if resp.Pools["func-2"].MaxConcurrency != 20 {
		t.Errorf("expected func-2 MaxConcurrency 20, got %d", resp.Pools["func-2"].MaxConcurrency)
	}
}

func TestAgentServer_EvictFunction_RemovesLocalDir(t *testing.T) {
	oldDir := config.FunctionsDir
	tempDir := t.TempDir()
	config.FunctionsDir = tempDir
	defer func() { config.FunctionsDir = oldDir }()

	funcDir := filepath.Join(tempDir, "some-func")
	if err := os.MkdirAll(funcDir, 0755); err != nil {
		t.Fatalf("failed to create dummy function dir: %v", err)
	}

	srv := NewAgentServer("n1", nil, &pool.PoolManager{}, nil)

	resp, err := srv.EvictFunction(context.Background(), &proto.EvictRequest{FuncName: "some-func"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !resp.Success {
		t.Error("expected evict to succeed")
	}

	if _, err := os.Stat(funcDir); !os.IsNotExist(err) {
		t.Errorf("expected directory %s to be removed, but it still exists or error: %v", funcDir, err)
	}
}

func TestAgentServer_NewAgentServer(t *testing.T) {
	pm := &pool.PoolManager{}
	srv := NewAgentServer("test-node-id", nil, pm, nil)

	if srv.nodeID != "test-node-id" {
		t.Errorf("expected nodeID 'test-node-id', got '%s'", srv.nodeID)
	}
	if srv.poolManager != pm {
		t.Error("expected poolManager to be set correctly")
	}
}
