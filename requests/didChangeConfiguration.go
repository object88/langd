package requests

import (
	"context"
	"encoding/json"

	"github.com/sourcegraph/jsonrpc2"
)

const (
	didChangeConfigurationMethod = "workspace/didChangeConfiguration"
)

type didChangeConfigurationHandler struct {
	requestBase
}

func createDidChangeConfigurationHandler(ctx context.Context, h *Handler, req *jsonrpc2.Request) requestHandler {
	rh := &didChangeConfigurationHandler{
		requestBase: createRequestBase(ctx, h, req),
	}

	return rh
}

func (rh *didChangeConfigurationHandler) preprocess(params *json.RawMessage) error {
	rh.h.log.Verbosef("Got '%s'\n", didChangeConfigurationMethod)

	var typedParams DidChangeConfigurationParams
	if err := json.Unmarshal(*params, &typedParams); err != nil {
		return err
	}

	rh.h.log.Verbosef("All changes: %#v\n", typedParams)

	return nil
}

func (rh *didChangeConfigurationHandler) work() error {
	return nil
}
