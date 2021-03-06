package client

import (
	"context"
	"fmt"
	"os"
	"os/exec"

	"github.com/object88/langd/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	address = "localhost:9876"
)

// Client is a wrapper for the gRPC client that handles connections
type Client struct {
	conn *grpc.ClientConn
	c    proto.LangdClient
}

// NewClient returns a new client
func NewClient() (*Client, error) {
	conn, err := grpc.Dial(address, grpc.WithInsecure())
	if err != nil {
		fmt.Printf("Failed to make dial.\n")
		return nil, err
	}

	c := proto.NewLangdClient(conn)

	return &Client{conn, c}, nil
}

// DestroyClient tears down the gRPC connection
func (c *Client) DestroyClient() {
	c.conn.Close()
}

// RequestLoad returns the load
func (c *Client) RequestLoad() error {
	load, err := c.c.Load(context.Background(), &proto.EmptyRequest{})
	if err != nil {
		s, ok := status.FromError(err)
		if !ok {
			return err
		}

		code := s.Code()
		if code == codes.Unavailable {
			return nil
		}

		return err
	}

	if load == nil {
		fmt.Printf("Nil load reported\n")
		return nil
	}

	fmt.Printf("CPU%%:\t%.02f\nMemory:\t%d MB\n", load.CpuLoad, load.MemoryLoad)

	return nil
}

// RequestNewService will spin up a new server-oriented process and
// return a new client with a connection to it
func (c *Client) RequestNewService() error {
	fmt.Printf("Starting...\n")
	cmd := exec.Command(os.Args[0], "serve")
	err := cmd.Start()
	if err != nil {
		fmt.Printf("Failed to spin up binary!\n")
		return err
	}

	fmt.Printf("Waiting for state to change away from %s.\n", c.conn.GetState().String())
	c.conn.WaitForStateChange(context.Background(), c.conn.GetState())
	fmt.Printf("Started.\n")

	return nil
}

// RequestShutdown sends a gRPC call to request the service process to shut down.
func (c *Client) RequestShutdown() error {
	_, err := c.c.Shutdown(context.Background(), &proto.EmptyRequest{})
	if err == nil {
		return nil
	}

	s, ok := status.FromError(err)
	if !ok {
		return err
	}

	code := s.Code()
	if code == codes.Unavailable {
		return nil
	}

	return err
}

// RequestStartup makes a no-op rGPC call to test whether the service is running
func (c *Client) RequestStartup() error {
	_, err := c.c.Startup(context.Background(), &proto.EmptyRequest{})
	if err == nil {
		return nil
	}

	return err
}
