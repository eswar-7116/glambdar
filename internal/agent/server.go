package agent

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/eswar-7116/glambdar/v3/internal/config"
	"github.com/eswar-7116/glambdar/v3/internal/docker"
	"github.com/eswar-7116/glambdar/v3/internal/functions"
	"github.com/eswar-7116/glambdar/v3/internal/pool"
	"github.com/eswar-7116/glambdar/v3/internal/storage"
	"github.com/eswar-7116/glambdar/v3/proto"
)

type AgentServer struct {
	proto.UnimplementedGlambdarAgentServer
	nodeID      string
	docker      *docker.Docker
	poolManager *pool.PoolManager
	store       storage.Storage
}

func NewAgentServer(nodeID string, d *docker.Docker, pm *pool.PoolManager, store storage.Storage) *AgentServer {
	return &AgentServer{
		nodeID:      nodeID,
		docker:      d,
		poolManager: pm,
		store:       store,
	}
}

func (s *AgentServer) Invoke(ctx context.Context, req *proto.InvokeRequest) (*proto.InvokeResponse, error) {
	fnReq := functions.InvokeRequest{
		Method:  req.GetMethod(),
		Headers: req.GetHeaders(),
		Body:    string(req.GetBody()),
	}

	resp, err := functions.Invoke(ctx, s.docker, req.GetFuncName(), fnReq)
	if err != nil {
		return nil, err
	}

	return &proto.InvokeResponse{
		StatusCode: int32(resp.StatusCode),
		Headers:    resp.Headers,
		Body:       resp.Body,
		ColdStart:  resp.ColdStart,
	}, nil
}

func (s *AgentServer) GetStatus(ctx context.Context, req *proto.StatusRequest) (*proto.StatusResponse, error) {
	nodeID := s.nodeID
	if nodeID == "" {
		if cfg, err := config.LoadConfig(); err == nil && cfg.NodeID != "" {
			nodeID = cfg.NodeID
		} else {
			nodeID = "agent-node"
		}
	}

	pools := make(map[string]*proto.PoolStatus)
	if s.poolManager != nil {
		statuses := s.poolManager.GetPoolStatuses()
		for k, v := range statuses {
			pools[k] = &proto.PoolStatus{
				IdleCount:      v.IdleCount,
				ActiveCount:    v.ActiveCount,
				MaxConcurrency: v.MaxConcurrency,
			}
		}
	}

	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	memoryAvailable := int64(m.Sys - m.Alloc)

	return &proto.StatusResponse{
		NodeId:          nodeID,
		Pools:           pools,
		CpuAvailable:    float64(runtime.NumCPU()),
		MemoryAvailable: memoryAvailable,
	}, nil
}

func (s *AgentServer) PreloadFunction(ctx context.Context, req *proto.PreloadRequest) (*proto.PreloadResponse, error) {
	funcName := req.GetFuncName()
	if funcName == "" {
		return &proto.PreloadResponse{Success: false, Message: "func_name is required"}, nil
	}

	if err := functions.EnsureCached(ctx, funcName); err != nil {
		return &proto.PreloadResponse{Success: false, Message: err.Error()}, nil
	}

	// Pre-warm a container for the function
	if s.poolManager != nil && s.docker != nil {
		p, err := s.poolManager.GetOrCreate(funcName, 10) // default max concurrency
		if err == nil {
			go pool.SpawnIdle(ctx, s.docker, config.FunctionsDir, funcName, p)
		}
	}

	return &proto.PreloadResponse{Success: true, Message: fmt.Sprintf("Function %s preloaded successfully", funcName)}, nil
}

func (s *AgentServer) EvictFunction(ctx context.Context, req *proto.EvictRequest) (*proto.EvictResponse, error) {
	funcName := req.GetFuncName()
	if funcName == "" {
		return &proto.EvictResponse{Success: false}, nil
	}

	if s.poolManager != nil && s.docker != nil {
		s.poolManager.DeletePool(ctx, s.docker, funcName)
	}

	funcDir := filepath.Join(config.FunctionsDir, funcName)
	_ = os.RemoveAll(funcDir)

	return &proto.EvictResponse{Success: true}, nil
}
