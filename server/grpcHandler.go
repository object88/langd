package server

import (
	"context"
	"fmt"
	"net/http"

	"github.com/object88/langd/proto"
	"google.golang.org/grpc"
)

// GrpcHandler keeps the gRPC Server reference
type GrpcHandler struct {
	S  *grpc.Server
	SM *http.Server
}

// Load returns the CPU and memory load
func (g *GrpcHandler) Load(ctx context.Context, _ *proto.EmptyRequest) (*proto.LoadReply, error) {
	load := &proto.LoadReply{
		CpuLoad:    0.75,
		MemoryLoad: 500,
	}
	return load, nil
}

// Shutdown stops the service process
func (g *GrpcHandler) Shutdown(ctx context.Context, _ *proto.EmptyRequest) (*proto.EmptyReply, error) {
	fmt.Printf("Requesting stop on JSON server\n")
	g.SM.Shutdown(ctx)

	fmt.Printf("Requesting stop on gRPC\n")
	g.S.GracefulStop()

	return &proto.EmptyReply{}, nil
}

// Startup is a no-op to start the service
func (g *GrpcHandler) Startup(_ context.Context, _ *proto.EmptyRequest) (*proto.EmptyReply, error) {
	return &proto.EmptyReply{}, nil
}
