package requests

import (
	"context"
	"encoding/json"
	"fmt"
	"go/parser"
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
		// return noopHandleFuncer
	}

	uri := string(typedParams.TextDocument.URI)
	fpath := strings.TrimPrefix(uri, "file://")

	rh.fpath = fpath

	return nil
}

func (rh *didCloseHandler) work() error {
	_, ok := rh.h.workspace.OpenedFiles[rh.fpath]
	if !ok {
		rh.h.log.Warnf("File %s is not opened\n", rh.fpath)
		return nil
	}

	rh.h.log.Debugf("File %s is open...\n", rh.fpath)
	delete(rh.h.workspace.OpenedFiles, rh.fpath)

	astFile, err := parser.ParseFile(rh.h.workspace.Fset, rh.fpath, nil, 0)
	if err != nil {
		rh.h.log.Errorf("Failed to parse file as provided by didOpen: %s\n", err.Error())
	}

	rh.h.workspace.Files[rh.fpath] = astFile

	rh.h.log.Debugf("File %s is closed\n", rh.fpath)

	return nil
}
