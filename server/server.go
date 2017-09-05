package server

import (
	"context"
	"encoding/json"
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

// Handler implements jsonrpc2.Handle
type Handler struct {
}

func (h *Handler) Handle(ctx context.Context, conn *jsonrpc2.Conn, req *jsonrpc2.Request) {
	fmt.Printf("Handling something something...\n")

	switch req.Method {
	case "initialize":
		fmt.Printf("Got initialize method\n")

		var params InitializeParams
		if err := json.Unmarshal(*req.Params, &params); err != nil {
			return
		}

		fmt.Printf("Got parameters: %#v\n", params)

		results := &InitializeResult{
			Capabilities: ServerCapabilities{
				TextDocumentSync: TextDocumentSyncOptions{
					Change: 0,
				},
				HoverProvider:                    false,
				CompletionProvider:               nil,
				SignatureHelpProvider:            nil,
				DefinitionProvider:               false,
				ReferencesProvider:               false,
				DocumentHighlightProvider:        false,
				DocumentSymbolProvider:           false,
				WorkspaceSymbolProvider:          false,
				CodeActionProvider:               false,
				CodeLensProvider:                 nil,
				DocumentFormattingProvider:       false,
				DocumentRangeFormattingProvider:  false,
				DocumentOnTypeFormattingProvider: nil,
				RenameProvider:                   false,
			},
		}

		err := conn.Reply(ctx, req.ID, results)
		if err != nil {
			fmt.Printf("Reply got error: %s\n", err.Error())
		}
		fmt.Printf("Responded to initialization request\n")

	case "initialized":
		// No-op.

	default:
		fmt.Printf("Unhandled method '%s'\n", req.Method)
	}
}

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
				fmt.Printf("Got connection\n")
				h := &Handler{}

				fmt.Printf("Created handler\n")
				os := jsonrpc2.NewBufferedStream(c, jsonrpc2.VSCodeObjectCodec{})

				fmt.Printf("Created object stream\n")
				jsonrpc2.NewConn(context.Background(), os, h)
				fmt.Printf("Attached handler\n")

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
