package requests

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"go/parser"
	"strings"

	"github.com/sourcegraph/jsonrpc2"
)

const (
	didOpenNotification = "textDocument/didOpen"
)

func (h *Handler) didOpen(ctx context.Context, req *jsonrpc2.Request) handleFuncer {
	var params DidOpenTextDocumentParams
	if err := json.Unmarshal(*req.Params, &params); err != nil {
		return noopHandleFuncer
	}

	fmt.Printf("Got parameters: %#v\n", params)

	uri := string(params.TextDocument.URI)
	fpath := strings.TrimPrefix(uri, "file://")

	buf := bytes.NewBufferString(params.TextDocument.Text)

	return func() {
		h.openedFiles[fpath] = buf

		if h.workspace == nil {
			h.log.Errorf("FAILED: Workspace doesn't exist on handler\n")
			return
		}

		astFile, err := parser.ParseFile(h.workspace.Fset, fpath, buf, 0)
		if err != nil {
			h.log.Errorf("Failed to parse file as provided by didOpen: %s\n", err.Error())
		}

		h.workspace.Files[fpath] = astFile

		h.log.Debugf("Shadowed file '%s'\n", fpath)

		// This is a notification, so no response necessary.
	}
}
