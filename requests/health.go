package requests

import (
	"context"
	"encoding/json"

	"github.com/sourcegraph/jsonrpc2"
)

const (
	healthMethod = "health/instant"
)

type healthHandler struct {
	requestBase

	result *Health
}

func createHealthHandler(ctx context.Context, h *Handler, req *jsonrpc2.Request) requestHandler {
	rh := &healthHandler{
		requestBase: createRequestBase(ctx, h, req, false),
	}

	return rh
}

func (rh *healthHandler) preprocess(params *json.RawMessage) error {
	rh.h.log.Verbosef("Got health method\n")
	return nil
}

func (rh *healthHandler) work() error {
	rh.result = &Health{
		CPU:    rh.h.load.CPU(),
		Memory: rh.h.load.Memory(),
	}

	return nil
}

func (rh *healthHandler) reply() (interface{}, error) {
	return rh.result, nil
}
