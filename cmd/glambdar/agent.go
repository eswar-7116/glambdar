package glambdar

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/eswar-7116/glambdar/v3/internal/agent"
	"github.com/eswar-7116/glambdar/v3/internal/cluster"
	"github.com/eswar-7116/glambdar/v3/internal/config"
	"github.com/eswar-7116/glambdar/v3/internal/pool"
	"github.com/eswar-7116/glambdar/v3/proto"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
)

var (
	agentGRPCPort   string
	agentNodeID     string
	agentRedisAddr  string
	agentWorkerPath string
)

var agentCmd = &cobra.Command{
	Use:   "agent",
	Short: "Run worker agent (data plane gRPC server)",
	RunE: func(cmd *cobra.Command, args []string) error {
		runAgent()
		return nil
	},
}

func init() {
	agentCmd.Flags().StringVar(&agentGRPCPort, "grpc-port", "9090", "gRPC server port")
	agentCmd.Flags().StringVar(&agentNodeID, "node-id", "", "Node ID (auto-generated if empty)")
	agentCmd.Flags().StringVar(&agentRedisAddr, "redis-addr", "", "Redis address for cluster coordination (e.g. localhost:6379)")
	agentCmd.Flags().StringVar(&agentWorkerPath, "worker-path", "", "Path to glambdar-worker.js")
}

func runAgent() {
	if err := config.InitAgentPaths(); err != nil {
		log.Printf("Notice initializing paths: %v", err)
	}

	cfg, _ := config.LoadConfig()
	if cfg == nil {
		cfg = &config.Config{}
	}

	if agentNodeID != "" {
		cfg.NodeID = agentNodeID
	}
	if cfg.NodeID == "" {
		cfg.NodeID = config.NodeID
	}
	if cfg.NodeID == "" {
		cfg.NodeID = "agent-" + fmt.Sprintf("%d", time.Now().UnixNano())
	}

	if agentRedisAddr != "" {
		cfg.RedisAddr = agentRedisAddr
	}

	wp := agentWorkerPath
	if wp != "" {
		config.DockerClient.WorkerPath = wp
	} else if config.DockerClient.WorkerPath == "" {
		if _, err := os.Stat("worker/glambdar-worker.js"); err == nil {
			config.DockerClient.WorkerPath, _ = filepath.Abs("worker/glambdar-worker.js")
		} else {
			config.DockerClient.WorkerPath = filepath.Join(config.ConfigDir, "worker", "glambdar-worker.js")
		}
	}

	pingCtx, pingCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer pingCancel()
	if err := config.DockerClient.Ping(pingCtx); err != nil {
		fmt.Println("Error: Docker daemon is not reachable. Please make sure Docker is running.")
		fmt.Printf("Details: %v\n", err)
		os.Exit(1)
	}

	agentServer := agent.NewAgentServer(cfg.NodeID, config.DockerClient, config.PoolManager, config.StorageClient)

	lis, err := net.Listen("tcp", ":"+agentGRPCPort)
	if err != nil {
		log.Fatalf("Failed to listen on gRPC port %s: %v", agentGRPCPort, err)
	}

	grpcServer := grpc.NewServer()
	proto.RegisterGlambdarAgentServer(grpcServer, agentServer)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if cfg.RedisAddr != "" {
		stateProvider := cluster.NewRedisStateProvider(cfg.RedisAddr)
		if err := stateProvider.Join(ctx); err != nil {
			log.Printf("Warning: failed to join Redis cluster state: %v", err)
		} else {
			log.Printf("Connected to Redis cluster state at %s", cfg.RedisAddr)
			go startHeartbeat(ctx, stateProvider, cfg.NodeID, ":"+agentGRPCPort, config.PoolManager)
			defer stateProvider.Leave(ctx)
		}
	}

	log.Printf("Glambdar Agent [%s] listening on gRPC port %s...", cfg.NodeID, agentGRPCPort)

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-stop
		log.Printf("Received signal %v, shutting down Glambdar Agent...", sig)
		grpcServer.GracefulStop()
		cancel()
	}()

	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("gRPC server failed: %v", err)
	}
}

func startHeartbeat(ctx context.Context, state cluster.StateProvider, nodeID, addr string, pm *pool.PoolManager) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			pools := make(map[string]cluster.PoolStatus)
			for k, v := range pm.GetPoolStatuses() {
				pools[k] = cluster.PoolStatus{
					IdleCount:      int(v.IdleCount),
					ActiveCount:    int(v.ActiveCount),
					MaxConcurrency: v.MaxConcurrency,
				}
			}

			status := &cluster.NodeStatus{
				NodeID:  nodeID,
				Address: addr,
				Pools:   pools,
			}
			if err := state.PublishStatus(ctx, status); err != nil {
				log.Printf("Failed to publish node status to Redis: %v", err)
			}
		case <-ctx.Done():
			return
		}
	}
}
