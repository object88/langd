package server

import (
	"context"
	"fmt"
	"log"
	"net"
	"sync"

	"github.com/object88/langd/proto"
	"github.com/object88/langd/requests"
	"github.com/sourcegraph/jsonrpc2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

const (
	port     = ":9876"
	jsonPort = ":9877"
)

// InitializeService runs for the lifespan of the server instance
func InitializeService() error {
	wg := &sync.WaitGroup{}
	wg.Add(2)

	go func() {
		socketService()
		wg.Done()
	}()

	go func() {
		grpcService()
		wg.Done()
	}()

	wg.Wait()

	fmt.Printf("Done.\n")

	return nil
}

func socketService() {
	fmt.Printf("JSON server starting\n")

	imf := requests.CreateIniterMapFactory()

	lis, err := net.Listen("tcp", jsonPort)
	if err != nil {
		fmt.Printf("Error listening on port %s: %s\n", jsonPort, err.Error())
		return
	}

	defer lis.Close()

	for {
		conn, err := lis.Accept()
		if err != nil {
			fmt.Printf("Got error on accept: %s\n", err.Error())
			break
		}

		go func(c net.Conn) {
			h := requests.NewHandler(imf)
			os := jsonrpc2.NewBufferedStream(c, jsonrpc2.VSCodeObjectCodec{})
			conn := jsonrpc2.NewConn(context.Background(), os, jsonrpc2.AsyncHandler(h))
			h.SetConn(conn)

			// // Shut down the connection.
			// fmt.Printf("Closing down...\n")
			// c.Close()
		}(conn)
	}
}

func grpcService() {
	fmt.Printf("gRPC server starting\n")

	lis, err := net.Listen("tcp", port)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
		return
	}

	s := grpc.NewServer()

	proto.RegisterLangdServer(s, &GrpcHandler{S: s, SM: nil})

	// Register reflection service on gRPC server.
	reflection.Register(s)
	err = s.Serve(lis)
	if err != nil {
		fmt.Printf("Got error when stopping grpc service:\n%s\n", err.Error())
	}

	fmt.Printf("gRPC server stopped\n")
}
