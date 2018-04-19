package server

import (
	"fmt"
	"net"
	"os"

	"github.com/object88/langd"
	"github.com/object88/langd/health"
	"github.com/object88/langd/log"
	"google.golang.org/grpc"
)

const (
	grpcPort = ":9876"
	jsonPort = ":9877"
)

// GrpcHandler keeps the gRPC Server reference
type GrpcHandler struct {
	S   *grpc.Server
	lis net.Listener
	srv *server
}

// SocketHandler has the http Server
type SocketHandler struct {
	lis net.Listener
	srv *server
}

type server struct {
	done   chan bool
	load   *health.Load
	loader langd.Loader
}

// InitializeService runs for the lifespan of the server instance
func InitializeService() error {
	l := log.CreateLog(os.Stdout)

	s := &server{
		done:   make(chan bool),
		load:   health.StartLoadMonitoring(),
		loader: langd.NewLoader(),
	}

	// s.loader.Start()

	// s.loader.Log = l

	grpcLis, err := net.Listen("tcp", grpcPort)
	if err != nil {
		l.Errorf("failed to listen: %v", err)
		return err
	}

	grpcHandler := &GrpcHandler{
		S:   grpc.NewServer(),
		lis: grpcLis,
		srv: s,
	}

	socketLis, err := net.Listen("tcp", jsonPort)
	if err != nil {
		l.Errorf("Error listening on port %s: %s\n", jsonPort, err.Error())
		return err
	}

	sockHandler := &SocketHandler{
		lis: socketLis,
		srv: s,
	}

	go sockHandler.socketService()
	go grpcHandler.grpcService()

	<-s.done

	grpcHandler.S.GracefulStop()
	grpcLis.Close()
	socketLis.Close()

	s.load.Close()

	fmt.Printf("Done.\n")

	return nil
}
