package requests

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/sourcegraph/jsonrpc2"
)

const (
	initializedNotification = "initialized"
)

type initializedHandler struct {
	requestBase
}

func createInitializedHandler(ctx context.Context, h *Handler, req *jsonrpc2.Request) requestHandler {
	rh := &initializedHandler{
		requestBase: createRequestBase(ctx, h, req, true),
	}

	return rh
}

func (rh *initializedHandler) preprocess(params *json.RawMessage) error {
	return nil
}

func (rh *initializedHandler) work() error {
	cParams := &[]ConfigurationParams{}
	result := []interface{}{}
	err := rh.h.conn.Call(context.Background(), "workspace/configuration", cParams, result)
	if err != nil {
		fmt.Printf("Error: %#v\n", err)
	}

	fmt.Printf("Result:\n\t%#v\n", result)

	return nil
}
