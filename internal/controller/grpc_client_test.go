package controller

import (
	"net"
	"testing"

	pb "github.com/eswar-7116/glambdar/v3/proto"
	"google.golang.org/grpc"
)

type dummyAgentServer struct {
	pb.UnimplementedGlambdarAgentServer
}

func TestGRPCClientPool(t *testing.T) {
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to listen: %v", err)
	}
	defer lis.Close()

	s := grpc.NewServer()
	pb.RegisterGlambdarAgentServer(s, &dummyAgentServer{})
	go func() {
		_ = s.Serve(lis)
	}()
	defer s.Stop()

	addr := lis.Addr().String()

	pool := NewGRPCClientPool()
	defer pool.CloseAll()

	client1, err := pool.GetClient(addr)
	if err != nil {
		t.Fatalf("failed to get client: %v", err)
	}
	if client1 == nil {
		t.Fatal("expected non-nil client")
	}

	// Verify caching
	client2, err := pool.GetClient(addr)
	if err != nil {
		t.Fatalf("failed to get cached client: %v", err)
	}
	if client1 != client2 {
		t.Fatalf("expected cached client instance to be identical, got %v vs %v", client1, client2)
	}

	pool.CloseAll()
	if len(pool.entries) != 0 {
		t.Fatalf("expected entries to be empty after CloseAll, got %d", len(pool.entries))
	}
}
