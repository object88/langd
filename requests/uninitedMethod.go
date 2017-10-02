package requests

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/sourcegraph/jsonrpc2"
)

type uninitedMethodHandler struct {
	requestBase
}

func createUninitedMethodHandler(ctx context.Context, h *Handler, req *jsonrpc2.Request) requestHandler {
	rh := &uninitedMethodHandler{
		requestBase: createRequestBase(ctx, h, req, false),
	}

	return rh
}

func (rh *uninitedMethodHandler) preprocess(params *json.RawMessage) error {
	return errors.New("NOPE")
}

func (rh *uninitedMethodHandler) work() error {
	return nil
}

func (rh *uninitedMethodHandler) reply() (interface{}, error) {
	return nil, errors.New("TODO")
}
