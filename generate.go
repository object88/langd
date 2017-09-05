package langd

import (
	"context"
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"github.com/object88/langd/proto"
	"google.golang.org/grpc"
)

// Generator keeps the gRPC Server reference
type Generator struct {
	S  *grpc.Server
	SM *http.Server
}

// GenerateUUID returns a string UUID
func (g *Generator) GenerateUUID(ctx context.Context, _ *proto.EmptyRequest) (*proto.UUIDReply, error) {
	u, err := uuid.NewRandom()
	if err != nil {
		return nil, err
	}

	resp := &proto.UUIDReply{
		Uuid: u.String(),
	}

	return resp, nil
}

// Shutdown stops the service process
func (g *Generator) Shutdown(ctx context.Context, _ *proto.EmptyRequest) (*proto.EmptyReply, error) {
	fmt.Printf("Requesting stop on JSON server\n")
	g.SM.Shutdown(ctx)

	fmt.Printf("Requesting stop on gRPC\n")
	g.S.GracefulStop()

	return &proto.EmptyReply{}, nil
}

// Startup is a no-op to start the service
func (g *Generator) Startup(_ context.Context, _ *proto.EmptyRequest) (*proto.EmptyReply, error) {
	return &proto.EmptyReply{}, nil
}
