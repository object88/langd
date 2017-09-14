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
		requestBase: createRequestBase(ctx, h, req.ID),
	}

	return ih
}

func (ih *initializedHandler) preprocess(p *json.RawMessage) {
	fmt.Printf("Got initialized method\n")
}

func (ih *initializedHandler) work() {
	fmt.Printf("Did initialized method\n")
}
