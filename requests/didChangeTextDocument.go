package requests

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/sourcegraph/jsonrpc2"
)

const (
	didChangeTextDocumentNotification = "textDocument/didChange"
)

type didChangeTextDocumentHandler struct {
	requestBase
	uri     string
	changes []TextDocumentContentChangeEvent
}

func createDidChangeTextDocumentHandler(ctx context.Context, h *Handler, req *jsonrpc2.Request) requestHandler {
	rh := &didChangeTextDocumentHandler{
		requestBase: createRequestBase(ctx, h, req, true),
	}

	return rh
}

func (rh *didChangeTextDocumentHandler) preprocess(params *json.RawMessage) error {
	var typedParams DidChangeTextDocumentParams
	if err := json.Unmarshal(*params, &typedParams); err != nil {
		return nil
	}

	uri := string(typedParams.TextDocument.URI)
	uri = strings.TrimPrefix(uri, "file://")

	rh.uri = uri
	rh.changes = typedParams.ContentChanges

	return nil
}

func (rh *didChangeTextDocumentHandler) work() error {
	// Wait in this until we have a way to validate the edits.
	uri := rh.uri

	for _, v := range rh.changes {
		if v.Range == nil || v.RangeLength == nil {
			rh.h.workspace.ReplaceFile(uri, v.Text)
		} else {
			rh.h.workspace.ChangeFile(uri, v.Range.Start.Line, v.Range.Start.Character, v.Range.End.Line, v.Range.End.Character, v.Text)
		}
	}

	return nil
}
