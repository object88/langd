package requests

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/sourcegraph/jsonrpc2"
)

const (
	didOpenNotification = "textDocument/didOpen"
)

type didOpenHandler struct {
	requestBase

	fpath string
	text  string
}

func createDidOpenHandler(ctx context.Context, h *Handler, req *jsonrpc2.Request) requestHandler {
	rh := &didOpenHandler{
		requestBase: createRequestBase(ctx, h, req, true),
	}

	return rh
}

func (rh *didOpenHandler) preprocess(params *json.RawMessage) error {
	var typedParams DidOpenTextDocumentParams
	if err := json.Unmarshal(*params, &typedParams); err != nil {
		return fmt.Errorf("Failed to unmarshal params")
	}

	fmt.Printf("Got parameters: %#v\n", typedParams)

	uri := string(typedParams.TextDocument.URI)
	fpath := strings.TrimPrefix(uri, "file://")

	rh.fpath = fpath
	rh.text = typedParams.TextDocument.Text
	return nil
}

func (rh *didOpenHandler) work() error {
	if rh.h.workspace == nil {
		return fmt.Errorf("FAILED: Workspace doesn't exist on handler")
	}

	return rh.h.workspace.OpenFile(rh.fpath, rh.text)
}
