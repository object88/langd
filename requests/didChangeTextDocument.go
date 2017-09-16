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

type didChangeTextDocumentHandler struct {
	requestBase
}

func createDidChangeTextDocumentHandler(ctx context.Context, h *Handler, req *jsonrpc2.Request) requestHandler {
	rh := &didChangeTextDocumentHandler{
		requestBase: createRequestBase(ctx, h, req),
	}

	return rh
}

func (rh *didChangeTextDocumentHandler) preprocess(params *json.RawMessage) error {
	var typedParams DidChangeTextDocumentParams
	if err := json.Unmarshal(*params, &typedParams); err != nil {
		return nil
	}

	rh.h.log.Verbosef("Got parameters: %#v\n", typedParams)

	uri := string(typedParams.TextDocument.URI)
	buf, ok := rh.h.openedFiles[uri]
	if !ok {
		fmt.Printf("File %s is not opened\n", uri)
		return nil
	}

	fmt.Printf("File %s is open, len %d\n", uri, buf.Len())
	return nil
}

func (rh *didChangeTextDocumentHandler) work() error {
	return nil
}
