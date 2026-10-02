package controller

import (
	"sync"

	pb "github.com/eswar-7116/glambdar/v3/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type clientEntry struct {
	conn   *grpc.ClientConn
	client pb.GlambdarAgentClient
}

// Manages persistent gRPC connections to agent nodes
type GRPCClientPool struct {
	mu      sync.RWMutex
	entries map[string]clientEntry
}

func NewGRPCClientPool() *GRPCClientPool {
	return &GRPCClientPool{
		entries: make(map[string]clientEntry),
	}
}

func (p *GRPCClientPool) GetClient(addr string) (pb.GlambdarAgentClient, error) {
	// Check if client already exists
	p.mu.RLock()
	if entry, ok := p.entries[addr]; ok {
		p.mu.RUnlock()
		return entry.client, nil
	}
	p.mu.RUnlock()

	// Write-lock
	p.mu.Lock()
	defer p.mu.Unlock()

	// Double-check with write-lock
	if entry, ok := p.entries[addr]; ok {
		return entry.client, nil
	}

	// Create new connection under write-lock
	conn, err := grpc.NewClient(addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, err
	}

	client := pb.NewGlambdarAgentClient(conn)
	p.entries[addr] = clientEntry{conn: conn, client: client}
	return client, nil
}

// Closes all open connections gracefully
func (p *GRPCClientPool) CloseAll() {
	p.mu.Lock()
	defer p.mu.Unlock()
	for addr, entry := range p.entries {
		entry.conn.Close()
		delete(p.entries, addr)
	}
}
