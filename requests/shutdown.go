package requests

import (
	"context"
	"encoding/json"

	"github.com/sourcegraph/jsonrpc2"
)

const (
	shutdownMethod = "shutdown"
)

type shutdownHandler struct {
	requestBase
}

func createShutdownHandler(ctx context.Context, h *Handler, req *jsonrpc2.Request) requestHandler {
	rh := &shutdownHandler{
		requestBase: createRequestBase(ctx, h, req, false),
	}

	return rh
}

func (rh *shutdownHandler) preprocess(params *json.RawMessage) error {
	rh.h.log.Debugf("Received shutdown request\n")
	return nil
}

func (rh *shutdownHandler) work() error {
	return nil
}

func (rh *shutdownHandler) reply() (interface{}, error) {
	return nil, nil
}
