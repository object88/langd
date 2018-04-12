package requests

import (
	"context"
)

const (
	logMessageNotification = "window/logMessage"
)

func logMessage(ctx context.Context, conn Conn, params *LogMessageParams) {
	conn.Notify(ctx, logMessageNotification, params)
}
