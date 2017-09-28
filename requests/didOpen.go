package requests

import (
	"context"
	"encoding/json"
	"fmt"
	"go/parser"
	"strings"

	"github.com/object88/rope"
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
		requestBase: createRequestBase(ctx, h, req),
	}

	return rh
}

func (rh *didOpenHandler) preprocess(params *json.RawMessage) error {
	var typedParams DidOpenTextDocumentParams
	if err := json.Unmarshal(*params, &typedParams); err != nil {
		return fmt.Errorf("Failed to unmarshal params")
		// return noopHandleFuncer
	}

	fmt.Printf("Got parameters: %#v\n", typedParams)

	uri := string(typedParams.TextDocument.URI)
	fpath := strings.TrimPrefix(uri, "file://")

	rh.fpath = fpath
	rh.text = typedParams.TextDocument.Text
	return nil
}

func (rh *didOpenHandler) work() error {
	rh.h.openedFiles[rh.fpath] = rope.CreateRope(rh.text)

	if rh.h.workspace == nil {
		return fmt.Errorf("FAILED: Workspace doesn't exist on handler")
	}

	astFile, err := parser.ParseFile(rh.h.workspace.Fset, rh.fpath, rh.text, 0)
	if err != nil {
		rh.h.log.Warnf("Failed to parse file as provided by didOpen: %s\n", err.Error())
	}

	rh.h.workspace.Files[rh.fpath] = astFile

	rh.h.log.Debugf("Shadowed file '%s'\n", rh.fpath)

	return nil
}
