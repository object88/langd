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
	ih := &initializedHandler{
		requestBase: createRequestBase(ctx, h, req),
	}

	return ih
}

func (ih *initializedHandler) preprocess(p *json.RawMessage) error {
	fmt.Printf("Got initialized method\n")
	return nil
}

func (ih *initializedHandler) work() error {
	fmt.Printf("Did initialized method\n")
	return nil
}
