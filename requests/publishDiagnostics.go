package requests

import (
	"context"
)

const (
	publishDiagnosticsNotification = "textDocument/publishDiagnostics"
)

func publishDiagnostics(ctx context.Context, conn Conn, params *PublishDiagnosticsParams) {
	conn.Notify(ctx, publishDiagnosticsNotification, params)
}
