package requests

import (
	"context"

	"github.com/sourcegraph/jsonrpc2"
)

const (
	shutdownMethod = "shutdown"
)

func (h *Handler) shutdown(ctx context.Context, conn *jsonrpc2.Conn, req *jsonrpc2.Request) {
	h.log.Debugf("Received shutdown request\n")
	h.workspace = nil
	conn.Reply(ctx, req.ID, nil)
}
