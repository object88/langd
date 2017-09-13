package requests

import (
	"context"

	"github.com/sourcegraph/jsonrpc2"
)

const (
	logMessageNotification = "window/logMessage"
)

func logMessage(ctx context.Context, conn *jsonrpc2.Conn, params *LogMessageParams) {
	conn.Notify(ctx, logMessageNotification, params)
}
