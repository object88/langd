package requests

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/sourcegraph/jsonrpc2"
)

const (
	didCloseNotification = "textDocument/didClose"
)

func (h *Handler) didClose(ctx context.Context, conn *jsonrpc2.Conn, req *jsonrpc2.Request) {
	var params DidCloseTextDocumentParams
	if err := json.Unmarshal(*req.Params, &params); err != nil {
		return
	}

	uri := string(params.TextDocument.URI)
	if opened, ok := h.openedFiles[uri]; !ok {
		fmt.Printf("File %s was never opened\n", uri)
	} else {
		if opened {
			fmt.Printf("File %s is opened\n", uri)
			h.openedFiles[uri] = false
		} else {
			fmt.Printf("File %s is not opened\n", uri)
		}
	}

}
