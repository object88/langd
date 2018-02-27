package server

import (
	"context"
	"fmt"

	"github.com/object88/langd/proto"
	"google.golang.org/grpc/reflection"
)

func (g *GrpcHandler) grpcService() {
	fmt.Printf("gRPC server starting\n")

	proto.RegisterLangdServer(g.S, g)

	// Register reflection service on gRPC server.
	reflection.Register(g.S)

	err := g.S.Serve(g.lis)
	if err != nil {
		fmt.Printf("Got error when stopping grpc service:\n%s\n", err.Error())
	}

	fmt.Printf("gRPC server stopped\n")
}

// Load returns the CPU and memory load
func (g *GrpcHandler) Load(_ context.Context, _ *proto.EmptyRequest) (*proto.LoadReply, error) {
	load := &proto.LoadReply{
		CpuLoad:    g.srv.load.CPU(),
		MemoryLoad: g.srv.load.Memory(),
	}
	return load, nil
}

// Shutdown stops the service process
func (g *GrpcHandler) Shutdown(ctx context.Context, _ *proto.EmptyRequest) (*proto.EmptyReply, error) {
	g.srv.done <- true
	return &proto.EmptyReply{}, nil
}

// Startup is a no-op to start the service
func (g *GrpcHandler) Startup(_ context.Context, _ *proto.EmptyRequest) (*proto.EmptyReply, error) {
	return &proto.EmptyReply{}, nil
}
