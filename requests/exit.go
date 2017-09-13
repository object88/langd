package requests

import (
	"context"

	"github.com/sourcegraph/jsonrpc2"
)

const (
	exitNotification = "exit"
)

func (h *Handler) exit(ctx context.Context, conn *jsonrpc2.Conn, req *jsonrpc2.Request) {
	// No response necessary.  Use exit status 0 if we have been asked to shut
	// down, otherwise 1.
}
