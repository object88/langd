package server

import (
	"context"
	"fmt"
	"net"

	"github.com/object88/langd/requests"
	"github.com/sourcegraph/jsonrpc2"
)

func (s *SocketHandler) socketService() {
	fmt.Printf("JSON server starting\n")

	for {
		conn, err := s.lis.Accept()
		if err != nil {
			fmt.Printf("Got error on accept: %s\n", err.Error())
			break
		}

		go func(c net.Conn) {
			h := requests.NewHandler(s.srv.load)
			os := jsonrpc2.NewBufferedStream(c, jsonrpc2.VSCodeObjectCodec{})
			conn := jsonrpc2.NewConn(context.Background(), os, h)
			h.SetConnection(conn)
		}(conn)
	}

	fmt.Printf("JSON server stopped\n")
}
