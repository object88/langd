package server

import (
	"context"
	"fmt"
	"log"
	"net"
	"sync"

	"github.com/object88/langd"
	"github.com/object88/langd/proto"
	"github.com/sourcegraph/jsonrpc2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

const (
	port     = ":9876"
	jsonPort = ":9877"
)

// InitializeService starts the service
func InitializeService() error {
	wg := &sync.WaitGroup{}

	go func() {
		fmt.Printf("JSON server starting\n")
		wg.Add(1)

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
				h := NewHandler()
				os := jsonrpc2.NewBufferedStream(c, jsonrpc2.VSCodeObjectCodec{})
				conn := jsonrpc2.NewConn(context.Background(), os, jsonrpc2.AsyncHandler(h))
				h.SetConn(conn)

				// // Shut down the connection.
				// fmt.Printf("Closing down...\n")
				// c.Close()
			}(conn)
		}

		wg.Done()
	}()

	lis, err := net.Listen("tcp", port)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
		return err
	}

	s := grpc.NewServer()

	go func() {
		fmt.Printf("gRPC server starting\n")
		wg.Add(1)

		proto.RegisterGeneratorServer(s, &langd.Generator{S: s, SM: nil})

		// Register reflection service on gRPC server.
		reflection.Register(s)
		s.Serve(lis)

		fmt.Printf("gRPC server stopped\n")
		wg.Done()
	}()

	wg.Wait()

	fmt.Printf("Done.\n")

	return nil
}
