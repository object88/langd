package requests

import (
	"context"
	"encoding/json"
	"go/parser"
	"strings"

	"github.com/sourcegraph/jsonrpc2"
)

const (
	didCloseNotification = "textDocument/didClose"
)

func (h *Handler) didClose(ctx context.Context, req *jsonrpc2.Request) handleFuncer {
	var params DidCloseTextDocumentParams
	if err := json.Unmarshal(*req.Params, &params); err != nil {
		return noopHandleFuncer
	}

	uri := string(params.TextDocument.URI)
	fpath := strings.TrimPrefix(uri, "file://")

	_, ok := h.openedFiles[fpath]
	if !ok {
		h.log.Warnf("File %s is not opened\n", fpath)
		return noopHandleFuncer
	}

	h.log.Debugf("File %s is open...\n", fpath)
	delete(h.openedFiles, fpath)

	astFile, err := parser.ParseFile(h.workspace.Fset, fpath, nil, 0)
	if err != nil {
		h.log.Errorf("Failed to parse file as provided by didOpen: %s\n", err.Error())
	}

	h.workspace.Files[fpath] = astFile

	h.log.Debugf("File %s is closed\n", fpath)
	return noopHandleFuncer
}
