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
		requestBase: createRequestBase(ctx, h, req.ID),
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
	_, ok := rh.h.openedFiles[rh.fpath]
	if !ok {
		rh.h.log.Warnf("File %s is not opened\n", rh.fpath)
		return nil
	}

	rh.h.log.Debugf("File %s is open...\n", rh.fpath)
	delete(rh.h.openedFiles, rh.fpath)

	astFile, err := parser.ParseFile(rh.h.workspace.Fset, rh.fpath, nil, 0)
	if err != nil {
		rh.h.log.Errorf("Failed to parse file as provided by didOpen: %s\n", err.Error())
	}

	rh.h.workspace.Files[rh.fpath] = astFile

	rh.h.log.Debugf("File %s is closed\n", rh.fpath)

	return nil
}

// func (h *Handler) didClose(ctx context.Context, req *jsonrpc2.Request) handleFuncer {
// 	var params DidCloseTextDocumentParams
// 	if err := json.Unmarshal(*req.Params, &params); err != nil {
// 		return noopHandleFuncer
// 	}

// 	uri := string(params.TextDocument.URI)
// 	fpath := strings.TrimPrefix(uri, "file://")

// 	_, ok := h.openedFiles[fpath]
// 	if !ok {
// 		h.log.Warnf("File %s is not opened\n", fpath)
// 		return noopHandleFuncer
// 	}

// 	h.log.Debugf("File %s is open...\n", fpath)
// 	delete(h.openedFiles, fpath)

// 	astFile, err := parser.ParseFile(h.workspace.Fset, fpath, nil, 0)
// 	if err != nil {
// 		h.log.Errorf("Failed to parse file as provided by didOpen: %s\n", err.Error())
// 	}

// 	h.workspace.Files[fpath] = astFile

// 	h.log.Debugf("File %s is closed\n", fpath)
// 	return noopHandleFuncer
// }
