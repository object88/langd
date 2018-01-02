package requests

import (
	"context"

	"github.com/sourcegraph/jsonrpc2"
)

const (
	publishDiagnosticsNotification = "textDocument/publishDiagnostics"
)

func publishDiagnostics(ctx context.Context, conn *jsonrpc2.Conn, params *PublishDiagnosticsParams) {
	conn.Notify(ctx, publishDiagnosticsNotification, params)
}
