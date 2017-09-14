package requests

import (
	"context"

	"github.com/sourcegraph/jsonrpc2"
)

const (
	shutdownMethod = "shutdown"
)

func (h *Handler) shutdown(ctx context.Context, req *jsonrpc2.Request) handleFuncer {
	h.log.Debugf("Received shutdown request\n")
	h.workspace = nil
	return func() {
		h.conn.Reply(ctx, req.ID, nil)
	}
}
