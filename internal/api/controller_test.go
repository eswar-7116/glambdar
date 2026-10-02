package api

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/eswar-7116/glambdar/v3/internal/cluster"
	"github.com/eswar-7116/glambdar/v3/internal/controller"
	pb "github.com/eswar-7116/glambdar/v3/proto"
	"github.com/gin-gonic/gin"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type mockStateProvider struct {
	healthyNodes []cluster.NodeStatus
}

func (m *mockStateProvider) PublishStatus(ctx context.Context, status *cluster.NodeStatus) error {
	return nil
}
func (m *mockStateProvider) GetNodesWithWarmPool(ctx context.Context, funcName string) ([]cluster.NodeStatus, error) {
	return nil, nil
}
func (m *mockStateProvider) GetHealthyNodes(ctx context.Context) ([]cluster.NodeStatus, error) {
	return m.healthyNodes, nil
}
func (m *mockStateProvider) Join(ctx context.Context) error  { return nil }
func (m *mockStateProvider) Leave(ctx context.Context) error { return nil }

type mockAgentServer struct {
	pb.UnimplementedGlambdarAgentServer
	invokeFn func(ctx context.Context, req *pb.InvokeRequest) (*pb.InvokeResponse, error)
}

func (s *mockAgentServer) Invoke(ctx context.Context, req *pb.InvokeRequest) (*pb.InvokeResponse, error) {
	if s.invokeFn != nil {
		return s.invokeFn(ctx, req)
	}
	return &pb.InvokeResponse{
		StatusCode: 200,
		Headers:    map[string]string{"Content-Type": "application/json"},
		Body:       []byte(`{"result":"hello from agent"}`),
	}, nil
}

func setupTestControllerRouter() *gin.Engine {
	r := gin.New()
	r.Use(gin.Recovery())
	registerInvokeRoutes(r)
	return r
}

func TestControllerMode_InvokeNoNodes(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockState := &mockStateProvider{healthyNodes: nil}
	router := cluster.NewRouter(mockState)
	pool := controller.NewGRPCClientPool()
	defer pool.CloseAll()

	SetControllerMode(router, pool, mockState)
	defer func() {
		controllerMode = false
		clusterRouter = nil
		grpcPool = nil
		stateProvider = nil
	}()

	engine := setupTestControllerRouter()

	req, _ := http.NewRequest("POST", "/invoke/my-fn", strings.NewReader(`{}`))
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected status 503 when no nodes available, got %d", w.Code)
	}
}

func TestControllerMode_InvokeSuccessAndErrors(t *testing.T) {
	gin.SetMode(gin.TestMode)

	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to listen: %v", err)
	}
	defer lis.Close()

	agentServer := &mockAgentServer{}
	grpcSrv := grpc.NewServer()
	pb.RegisterGlambdarAgentServer(grpcSrv, agentServer)
	go func() {
		_ = grpcSrv.Serve(lis)
	}()
	defer grpcSrv.Stop()

	agentAddr := lis.Addr().String()

	mockState := &mockStateProvider{
		healthyNodes: []cluster.NodeStatus{
			{NodeID: "agent-1", Address: agentAddr},
		},
	}
	router := cluster.NewRouter(mockState)
	pool := controller.NewGRPCClientPool()
	defer pool.CloseAll()

	SetControllerMode(router, pool, mockState)
	defer func() {
		controllerMode = false
		clusterRouter = nil
		grpcPool = nil
		stateProvider = nil
	}()

	engine := setupTestControllerRouter()

	// Success test
	agentServer.invokeFn = func(ctx context.Context, req *pb.InvokeRequest) (*pb.InvokeResponse, error) {
		return &pb.InvokeResponse{
			StatusCode: 200,
			Headers:    map[string]string{"X-Test": "Passed"},
			Body:       []byte(`{"greeting":"hello world"}`),
		}, nil
	}

	req, _ := http.NewRequest("POST", "/invoke/test-fn", strings.NewReader(`{}`))
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}
	if w.Header().Get("X-Test") != "Passed" {
		t.Fatalf("expected X-Test header 'Passed', got %s", w.Header().Get("X-Test"))
	}
	var respBody map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &respBody); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	if respBody["greeting"] != "hello world" {
		t.Fatalf("expected greeting 'hello world', got %s", respBody["greeting"])
	}

	// Rate limit
	agentServer.invokeFn = func(ctx context.Context, req *pb.InvokeRequest) (*pb.InvokeResponse, error) {
		return nil, status.Error(codes.ResourceExhausted, "rate limit exceeded")
	}

	req, _ = http.NewRequest("POST", "/invoke/test-fn", strings.NewReader(`{}`))
	w = httptest.NewRecorder()
	engine.ServeHTTP(w, req)

	if w.Code != http.StatusTooManyRequests {
		t.Fatalf("expected status 429, got %d", w.Code)
	}

	// Not Found
	agentServer.invokeFn = func(ctx context.Context, req *pb.InvokeRequest) (*pb.InvokeResponse, error) {
		return nil, status.Error(codes.NotFound, "function not found")
	}

	req, _ = http.NewRequest("POST", "/invoke/test-fn", strings.NewReader(`{}`))
	w = httptest.NewRecorder()
	engine.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d", w.Code)
	}
}
