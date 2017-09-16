package requests

import (
	"context"
	"encoding/json"

	"github.com/sourcegraph/jsonrpc2"
)

const (
	exitNotification = "exit"
)

type exitHandler struct {
	requestBase
}

func createEditHandler(ctx context.Context, h *Handler, req *jsonrpc2.Request) requestHandler {
	rh := &exitHandler{
		requestBase: createRequestBase(ctx, h, req),
	}

	return rh
}

func (rh *exitHandler) preprocess(params *json.RawMessage) error {
	rh.h.log.Debugf("Received exit request\n")
	return nil
}

func (rh *exitHandler) work() error {
	return nil
}
