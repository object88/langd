package requests

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/sourcegraph/jsonrpc2"
)

const (
	didChangeTextDocumentNotification = "textDocument/didChange"
)

func (h *Handler) didChangeTextDocument(ctx context.Context, req *jsonrpc2.Request) handleFuncer {
	var params DidChangeTextDocumentParams
	if err := json.Unmarshal(*req.Params, &params); err != nil {
		return noopHandleFuncer
	}

	h.log.Verbosef("Got parameters: %#v\n", params)

	uri := string(params.TextDocument.URI)
	buf, ok := h.openedFiles[uri]
	if !ok {
		fmt.Printf("File %s is not opened\n", uri)
		return noopHandleFuncer
	}

	fmt.Printf("File %s is open, len %d\n", uri, buf.Len())
	return noopHandleFuncer
}
