package requests

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/sourcegraph/jsonrpc2"
)

const (
	didCloseNotification = "textDocument/didClose"
)

type didCloseHandler struct {
	requestBase
	fpath string
}

func createDidCloseHandler(ctx context.Context, h *Handler, req *jsonrpc2.Request) requestHandler {
	rh := &didCloseHandler{
		requestBase: createRequestBase(ctx, h, req, true),
	}

	return rh
}

func (rh *didCloseHandler) preprocess(params *json.RawMessage) error {
	var typedParams DidCloseTextDocumentParams
	if err := json.Unmarshal(*params, &typedParams); err != nil {
		return fmt.Errorf("Failed to unmarshal params")
	}

	uri := string(typedParams.TextDocument.URI)
	fpath := strings.TrimPrefix(uri, "file://")

	rh.fpath = fpath

	return nil
}

func (rh *didCloseHandler) work() error {
	return rh.h.workspace.CloseFile(rh.fpath)
}
