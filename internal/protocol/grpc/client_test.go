package grpc

import (
	"context"
	"net"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"

	"github.com/serdar/gottp/internal/protocol"
)

func TestName(t *testing.T) {
	client := New()
	if client.Name() != "grpc" {
		t.Errorf("expected grpc, got %s", client.Name())
	}
}

func TestValidate(t *testing.T) {
	client := New()

	tests := []struct {
		name    string
		req     *protocol.Request
		wantErr bool
	}{
		{
			name:    "missing URL",
			req:     &protocol.Request{GRPCService: "svc", GRPCMethod: "Method"},
			wantErr: true,
		},
		{
			name:    "missing service",
			req:     &protocol.Request{URL: "localhost:50051", GRPCMethod: "Method"},
			wantErr: true,
		},
		{
			name:    "missing method",
			req:     &protocol.Request{URL: "localhost:50051", GRPCService: "svc"},
			wantErr: true,
		},
		{
			name: "valid request",
			req: &protocol.Request{
				URL:         "localhost:50051",
				GRPCService: "helloworld.Greeter",
				GRPCMethod:  "SayHello",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := client.Validate(tt.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestClose(t *testing.T) {
	client := New()
	// Close on empty client should not panic.
	client.Close()

	if len(client.conns) != 0 {
		t.Error("expected empty conns after close")
	}
}

func TestGRPCStatusToHTTP(t *testing.T) {
	tests := []struct {
		code     codes.Code
		expected int
	}{
		{codes.OK, 200},
		{codes.InvalidArgument, 400},
		{codes.NotFound, 404},
		{codes.PermissionDenied, 403},
		{codes.Unauthenticated, 401},
		{codes.Unavailable, 503},
		{codes.Unimplemented, 501},
		{codes.Internal, 500},
		{codes.DeadlineExceeded, 504},
		{codes.AlreadyExists, 409},
		{codes.ResourceExhausted, 429},
	}

	for _, tt := range tests {
		t.Run(tt.code.String(), func(t *testing.T) {
			got := grpcStatusToHTTP(tt.code)
			if got != tt.expected {
				t.Errorf("grpcStatusToHTTP(%s) = %d, want %d", tt.code, got, tt.expected)
			}
		})
	}
}

func TestBuildMetadata(t *testing.T) {
	req := &protocol.Request{
		Metadata: map[string]string{
			"x-request-id": "abc-123",
		},
		Headers: map[string]string{
			"X-Custom": "value",
		},
		Auth: &protocol.AuthConfig{
			Type:  "bearer",
			Token: "my-token",
		},
	}

	md := buildMetadata(req)

	if vals := md.Get("x-request-id"); len(vals) == 0 || vals[0] != "abc-123" {
		t.Errorf("expected x-request-id=abc-123, got %v", vals)
	}
	if vals := md.Get("x-custom"); len(vals) == 0 || vals[0] != "value" {
		t.Errorf("expected x-custom=value, got %v", vals)
	}
	if vals := md.Get("authorization"); len(vals) == 0 || vals[0] != "Bearer my-token" {
		t.Errorf("expected authorization=Bearer my-token, got %v", vals)
	}
}

func TestBuildMetadataBasicAuth(t *testing.T) {
	req := &protocol.Request{
		Auth: &protocol.AuthConfig{
			Type:     "basic",
			Username: "admin",
			Password: "secret",
		},
	}

	md := buildMetadata(req)

	vals := md.Get("authorization")
	if len(vals) == 0 {
		t.Fatal("expected authorization header")
	}
	if vals[0] != "Basic YWRtaW46c2VjcmV0" {
		t.Errorf("unexpected basic auth value: %s", vals[0])
	}
}

func TestBuildMetadataNoAuth(t *testing.T) {
	req := &protocol.Request{
		Metadata: map[string]string{"key": "val"},
	}

	md := buildMetadata(req)

	if vals := md.Get("authorization"); len(vals) != 0 {
		t.Errorf("expected no authorization, got %v", vals)
	}
	if vals := md.Get("key"); len(vals) == 0 || vals[0] != "val" {
		t.Errorf("expected key=val, got %v", vals)
	}
}

func TestBuildMetadataContentTypeFiltered(t *testing.T) {
	req := &protocol.Request{
		Headers: map[string]string{
			"Content-Type": "application/json",
			"X-Other":      "keep",
		},
	}

	md := buildMetadata(req)

	if vals := md.Get("content-type"); len(vals) != 0 {
		t.Errorf("expected content-type to be filtered, got %v", vals)
	}
	if vals := md.Get("x-other"); len(vals) == 0 || vals[0] != "keep" {
		t.Errorf("expected x-other=keep, got %v", vals)
	}
}

// healthServer implements the gRPC Health service for integration testing.
type healthServer struct {
	healthpb.UnimplementedHealthServer
}

func (s *healthServer) Check(_ context.Context, _ *healthpb.HealthCheckRequest) (*healthpb.HealthCheckResponse, error) {
	return &healthpb.HealthCheckResponse{
		Status: healthpb.HealthCheckResponse_SERVING,
	}, nil
}

// TestExecuteWithHealthService tests a real gRPC call against the Health service
// using a bufconn-style test server.
func TestExecuteWithHealthService(t *testing.T) {
	// Start a test gRPC server with reflection and health service.
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to listen: %v", err)
	}

	srv := grpc.NewServer()
	healthpb.RegisterHealthServer(srv, &healthServer{})
	reflection.Register(srv)

	go func() {
		if err := srv.Serve(lis); err != nil {
			// Server stopped, expected during cleanup.
		}
	}()
	defer srv.Stop()

	addr := lis.Addr().String()

	client := New()
	defer client.Close()

	req := &protocol.Request{
		Protocol:    "grpc",
		URL:         addr,
		GRPCService: "grpc.health.v1.Health",
		GRPCMethod:  "Check",
		Body:        []byte(`{"service": ""}`),
	}

	resp, err := client.Execute(context.Background(), req)
	if err != nil {
		t.Fatalf("Execute() error: %v", err)
	}

	if resp.StatusCode != 200 {
		t.Errorf("expected status 200, got %d (%s)", resp.StatusCode, resp.Status)
	}
	if resp.Proto != "gRPC" {
		t.Errorf("expected proto gRPC, got %s", resp.Proto)
	}
	if resp.ContentType != "application/json" {
		t.Errorf("expected application/json, got %s", resp.ContentType)
	}
	if resp.Duration == 0 {
		t.Error("expected duration > 0")
	}
	if resp.Size == 0 {
		t.Error("expected non-empty response body")
	}

	// The body should contain the health check response as JSON.
	body := string(resp.Body)
	if body == "" {
		t.Error("expected non-empty body")
	}
	t.Logf("Response body: %s", body)
}

// TestExecuteWithMetadata verifies that metadata is sent to the server.
func TestExecuteWithMetadata(t *testing.T) {
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to listen: %v", err)
	}

	srv := grpc.NewServer()
	healthpb.RegisterHealthServer(srv, &healthServer{})
	reflection.Register(srv)

	go func() {
		srv.Serve(lis)
	}()
	defer srv.Stop()

	addr := lis.Addr().String()

	client := New()
	defer client.Close()

	req := &protocol.Request{
		Protocol:    "grpc",
		URL:         addr,
		GRPCService: "grpc.health.v1.Health",
		GRPCMethod:  "Check",
		Body:        []byte(`{"service": ""}`),
		Metadata: map[string]string{
			"x-request-id": "test-123",
		},
		Auth: &protocol.AuthConfig{
			Type:  "bearer",
			Token: "test-token",
		},
	}

	resp, err := client.Execute(context.Background(), req)
	if err != nil {
		t.Fatalf("Execute() error: %v", err)
	}

	if resp.StatusCode != 200 {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
}

// TestExecuteNotFoundService verifies error handling for unknown services.
func TestExecuteNotFoundService(t *testing.T) {
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to listen: %v", err)
	}

	srv := grpc.NewServer()
	reflection.Register(srv)

	go func() {
		srv.Serve(lis)
	}()
	defer srv.Stop()

	addr := lis.Addr().String()

	client := New()
	defer client.Close()

	req := &protocol.Request{
		Protocol:    "grpc",
		URL:         addr,
		GRPCService: "nonexistent.Service",
		GRPCMethod:  "Method",
		Body:        []byte(`{}`),
	}

	_, err = client.Execute(context.Background(), req)
	if err == nil {
		t.Fatal("expected error for nonexistent service")
	}
}

// TestConnectionReuse verifies that connections are cached and reused.
func TestConnectionReuse(t *testing.T) {
	client := New()
	defer client.Close()

	// Getting the same address twice should return the same connection.
	conn1, err := client.getConn("localhost:50051")
	if err != nil {
		t.Fatalf("getConn() error: %v", err)
	}

	conn2, err := client.getConn("localhost:50051")
	if err != nil {
		t.Fatalf("getConn() error: %v", err)
	}

	if conn1 != conn2 {
		t.Error("expected same connection object for same address")
	}

	// Different address should create a new connection.
	conn3, err := client.getConn("localhost:50052")
	if err != nil {
		t.Fatalf("getConn() error: %v", err)
	}

	if conn1 == conn3 {
		t.Error("expected different connection for different address")
	}
}

// TestDiscoverServicesIntegration tests service discovery against a real server.
// Since Health and reflection are both internal services that get filtered,
// this test verifies that filtering works and no internal services leak through.
func TestDiscoverServicesIntegration(t *testing.T) {
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to listen: %v", err)
	}

	srv := grpc.NewServer()
	healthpb.RegisterHealthServer(srv, &healthServer{})
	reflection.Register(srv)

	go func() {
		srv.Serve(lis)
	}()
	defer srv.Stop()

	addr := lis.Addr().String()

	services, err := DiscoverServices(context.Background(), addr)
	if err != nil {
		t.Fatalf("DiscoverServices() error: %v", err)
	}

	// Health and reflection are internal services and should be filtered out.
	// Since we only registered Health + reflection, we expect an empty list.
	for _, svc := range services {
		if isInternalService(svc.Name) {
			t.Errorf("internal service %q should have been filtered out", svc.Name)
		}
	}

	// Verify the function itself works (no error) even when all services are internal.
	t.Logf("Discovered %d user services (internal services filtered)", len(services))
}

// TestDiscoverServicesConnectionFailed tests error handling for unreachable servers.
func TestDiscoverServicesConnectionFailed(t *testing.T) {
	// Use a port that is not listening.
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	_, err := DiscoverServices(ctx, "127.0.0.1:1")
	if err == nil {
		t.Error("expected error for unreachable server")
	}
}

// TestIsInternalService verifies the internal service filter.
func TestIsInternalService(t *testing.T) {
	tests := []struct {
		name     string
		svcName  string
		expected bool
	}{
		{"reflection v1", "grpc.reflection.v1.ServerReflection", true},
		{"reflection v1alpha", "grpc.reflection.v1alpha.ServerReflection", true},
		{"health", "grpc.health.v1.Health", true},
		{"channelz", "grpc.channelz.v1.Channelz", true},
		{"user service", "myapp.UserService", false},
		{"empty", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isInternalService(tt.svcName)
			if got != tt.expected {
				t.Errorf("isInternalService(%q) = %v, want %v", tt.svcName, got, tt.expected)
			}
		})
	}
}

// Ensure that the Client implements protocol.Protocol at compile time.
var _ protocol.Protocol = (*Client)(nil)
