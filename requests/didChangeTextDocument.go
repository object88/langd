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

func (h *Handler) didChangeTextDocument(ctx context.Context, conn *jsonrpc2.Conn, req *jsonrpc2.Request) {
	var params DidChangeTextDocumentParams
	if err := json.Unmarshal(*req.Params, &params); err != nil {
		return
	}

	h.log.Verbosef("Got parameters: %#v\n", params)

	uri := string(params.TextDocument.URI)
	if opened, ok := h.openedFiles[uri]; !ok {
		fmt.Printf("File %s was never opened\n", uri)
	} else {
		if opened {
			fmt.Printf("File %s is opene\n", uri)
		} else {
			fmt.Printf("File %s is not opened\n", uri)
		}
	}
}
