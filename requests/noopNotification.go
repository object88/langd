package requests

import (
	"context"
	"encoding/json"

	"github.com/sourcegraph/jsonrpc2"
)

const (
	initializedNotification = "initialized"
	willSaveNotification    = "textDocument/willSave"
)

type noopNotificationHandler struct {
	requestBase
}

func createNoopNotificationHandler(ctx context.Context, h *Handler, req *jsonrpc2.Request) requestHandler {
	rh := &noopNotificationHandler{
		requestBase: createRequestBase(ctx, h, req),
	}

	return rh
}

func (rh *noopNotificationHandler) preprocess(params *json.RawMessage) error {
	return nil
}

func (rh *noopNotificationHandler) work() error {
	return nil
}
